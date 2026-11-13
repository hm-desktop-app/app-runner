package main

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"app-runner/internal/launcher"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type LauncherService struct {
	store     *launcher.Store
	app       *application.App
	window    *application.WebviewWindow
	updating  atomic.Bool
	hotkeyMu  sync.Mutex
	hotkeyErr string
	// modalDialogs 记录当前打开的模态原生对话框数量，供失去激活的处理逻辑跳过误判。
	modalDialogs atomic.Int32
}

// nativeDialogOpen reports whether a modal native dialog currently owns the activation.
func (s *LauncherService) nativeDialogOpen() bool { return s.modalDialogs.Load() > 0 }

func (s *LauncherService) Scan() (launcher.Snapshot, error) { return s.store.Scan() }
func (s *LauncherService) Save(id string, profile launcher.Profile) (launcher.App, error) {
	return s.store.Save(id, profile)
}
func (s *LauncherService) CheckEntry(id, entry string) bool {
	a, err := s.store.Get(id)
	if err != nil {
		return false
	}
	_, err = launcher.ResolveEntry(a, entry)
	return err == nil
}
func (s *LauncherService) Launch(id string) (launcher.App, error) {
	a, err := s.store.Get(id)
	if err != nil {
		return a, err
	}
	if err = launcher.Launch(a); err != nil {
		return a, err
	}
	if err = s.store.RecordLaunch(id); err != nil {
		s.modalDialogs.Add(1)
		s.app.Dialog.Warning().SetTitle("使用记录").SetMessage("应用已启动，但无法保存使用记录：" + err.Error()).AttachToWindow(s.window).Show()
		s.modalDialogs.Add(-1)
	}
	return s.store.Get(id)
}
func (s *LauncherService) OpenDirectory(id string) error {
	a, err := s.store.Get(id)
	if err != nil {
		return err
	}
	return launcher.OpenDirectory(a.Directory)
}
func (s *LauncherService) OpenLibrary() error {
	return launcher.OpenDirectory(s.store.Library())
}
func (s *LauncherService) EditProfile(id string) error {
	a, err := s.store.Get(id)
	if err != nil {
		return err
	}
	return launcher.EditProfile(filepath.Join(a.Directory, "app.toml"))
}
func (s *LauncherService) OpenLink(address string) error {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("请输入有效的 HTTP 或 HTTPS 链接")
	}
	return s.app.Browser.OpenURL(address)
}
func (s *LauncherService) ShowError(message string) {
	s.modalDialogs.Add(1)
	defer s.modalDialogs.Add(-1)
	s.app.Dialog.Error().SetTitle("操作失败").SetMessage(message).AttachToWindow(s.window).Show()
}
func (s *LauncherService) Confirm(message string) bool {
	s.modalDialogs.Add(1)
	defer s.modalDialogs.Add(-1)
	result := make(chan bool, 1)
	dialog := s.app.Dialog.Question().SetTitle("App Runner").SetMessage(message).AttachToWindow(s.window)
	// Wails maps native Yes/No response IDs to these labels; Windows localises captions.
	dialog.AddButton("Yes").OnClick(func() { result <- true })
	dialog.AddButton("No").SetAsDefault().SetAsCancel().OnClick(func() { result <- false })
	dialog.Show()
	select {
	case answer := <-result:
		return answer
	default:
		return false
	}
}
func (s *LauncherService) Update(id, source string) (*launcher.UpdateResult, error) {
	if !s.updating.CompareAndSwap(false, true) {
		return nil, errors.New("正在部署或安装更新，请稍候")
	}
	defer s.updating.Store(false)
	result, err := s.store.Update(id, source)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (s *LauncherService) DeleteBackup(id string) error {
	if !s.updating.CompareAndSwap(false, true) {
		return errors.New("正在部署或安装更新，请稍候")
	}
	defer s.updating.Store(false)
	return s.store.DeleteBackup(id)
}
func (s *LauncherService) PickSource(kind string) (string, error) {
	s.modalDialogs.Add(1)
	defer s.modalDialogs.Add(-1)
	dialog := s.app.Dialog.OpenFile().AttachToWindow(s.window)
	switch kind {
	case "directory":
		dialog.SetTitle("选择软件目录").CanChooseDirectories(true).CanChooseFiles(false)
	case "archive":
		dialog.SetTitle("选择 ZIP / 7z 压缩包").CanChooseDirectories(false).CanChooseFiles(true).AddFilter("压缩包 (*.zip;*.7z)", "*.zip;*.7z")
	default:
		return "", errors.New("请选择目录或压缩包")
	}
	return dialog.PromptForSingleSelection()
}
func (s *LauncherService) PrepareNewApp(source, name string) (launcher.NewAppInfo, error) {
	if !s.updating.CompareAndSwap(false, true) {
		return launcher.NewAppInfo{}, errors.New("正在部署或安装更新，请稍候")
	}
	defer s.updating.Store(false)
	return s.store.PrepareNewApp(source, name)
}
func (s *LauncherService) CommitNewApp(token string, profile launcher.Profile) (launcher.App, error) {
	if !s.updating.CompareAndSwap(false, true) {
		return launcher.App{}, errors.New("正在部署或安装更新，请稍候")
	}
	defer s.updating.Store(false)
	return s.store.CommitNewApp(token, profile)
}
func (s *LauncherService) CheckNewEntry(token, entry string) bool {
	return s.store.CheckNewEntry(token, entry)
}
func (s *LauncherService) DiscardNewApp(token string) error { return s.store.DiscardNewApp(token) }

func (s *LauncherService) SetTheme(dark bool) {
	if s.window == nil {
		return
	}
	if dark {
		s.window.SetBackgroundColour(application.NewRGB(23, 27, 33))
	} else {
		s.window.SetBackgroundColour(application.NewRGB(255, 255, 255))
	}
	application.InvokeSync(func() { applyNativeTheme(s.window, dark) })
}

func (s *LauncherService) SetCategory(id, category string) (launcher.App, error) {
	return s.store.SetCategory(id, category)
}

func (s *LauncherService) RenameCategory(oldName, newName string) ([]launcher.App, error) {
	return s.store.RenameCategory(oldName, newName)
}
func (s *LauncherService) DeleteCategory(name string) ([]launcher.App, error) {
	return s.store.DeleteCategory(name)
}

func (s *LauncherService) HideWindow() {
	if s.window == nil {
		return
	}
	s.window.Hide()
}

func (s *LauncherService) QuitApp() {
	if s.app == nil {
		return
	}
	s.app.Quit()
}

type Settings struct {
	Hotkey      string `json:"hotkey"`
	HotkeyError string `json:"hotkeyError"`
	Library     string `json:"library"`
}

// hotkeyAccelerator is the system-wide shortcut that reveals or hides the main window.
const hotkeyAccelerator = "Ctrl+Alt+Space"

func (s *LauncherService) Settings() Settings {
	s.hotkeyMu.Lock()
	hotkeyErr := s.hotkeyErr
	s.hotkeyMu.Unlock()
	return Settings{Hotkey: hotkeyAccelerator, HotkeyError: hotkeyErr, Library: s.store.Library()}
}

func (s *LauncherService) setHotkeyError(message string) {
	s.hotkeyMu.Lock()
	defer s.hotkeyMu.Unlock()
	s.hotkeyErr = message
}
