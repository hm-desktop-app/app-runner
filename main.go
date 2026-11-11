package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"app-runner/internal/launcher"
	"app-runner/internal/singleinstance"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	options, err := launcher.ParseOptions(os.Args[1:], os.Getenv("USER_LOCAL_APP"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		selftest := false
		for _, arg := range os.Args[1:] {
			if arg == "--selftest" {
				selftest = true
			}
		}
		if selftest {
			writeReport(map[string]string{"error": err.Error()})
		} else {
			launcher.ShowStartupError(err.Error())
		}
		os.Exit(2)
	}
	if options.Help {
		fmt.Println("App Runner\n  --app-dir <路径>   指定应用库（优先于 USER_LOCAL_APP）\n  --selftest         扫描应用库并写入 selfcheck/selftest.json\n  --help             显示帮助")
		return
	}
	store := launcher.NewStore(options.Library)
	if options.SelfTest {
		snapshot, err := store.Scan()
		if err != nil {
			writeReport(map[string]string{"error": err.Error()})
			os.Exit(1)
		}
		writeReport(snapshot)
		return
	}
	primary, err := singleinstance.Acquire()
	if err != nil {
		launcher.ShowStartupError("无法检查单实例状态：" + err.Error())
		os.Exit(1)
	}
	if primary == nil {
		return
	}
	defer primary.Close()
	defer store.CleanupImports()
	service := &LauncherService{store: store}
	var window *application.WebviewWindow
	show := func() {
		if window != nil {
			window.Show()
			window.Restore()
			window.Focus()
		}
	}
	toggle := func() {
		if window != nil && window.IsVisible() && window.IsFocused() {
			window.Hide()
			return
		}
		show()
	}
	app := application.New(application.Options{
		Name: "App Runner", Description: "本地应用管理与启动器", Icon: icon,
		Services:   []application.Service{application.NewService(service)},
		Assets:     application.AssetOptions{Handler: store.IconHandler(application.AssetFileServerFS(assets))},
		ShouldQuit: func() bool { return !service.updating.Load() },
	})
	service.app = app
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "main", Title: "App Runner", Width: 920, Height: 640, MinWidth: 740, MinHeight: 470,
		Frameless: true, InitialPosition: application.WindowCentered, DevToolsEnabled: false,
		BackgroundColour: application.NewRGB(255, 255, 255), URL: "/",
	})
	service.window = window
	ready := make(chan struct{})
	var watchOnce, hotkeyOnce sync.Once
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		log.Print("App Runner: native window ready")
		// Register after startup so the OS rejection (for example, the shortcut is
		// already owned by another application) is returned here instead of being
		// deferred to the application error handler.
		hotkeyOnce.Do(func() {
			if err := app.GlobalShortcut.Register(hotkeyAccelerator, toggle); err != nil {
				service.setHotkeyError(err.Error())
				log.Printf("App Runner: 全局热键 %s 注册失败：%v", hotkeyAccelerator, err)
			}
		})
		watchOnce.Do(func() { close(ready) })
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { window.Hide(); e.Cancel() })
	tray := app.SystemTray.New()
	tray.SetIcon(icon)
	tray.SetTooltip("App Runner")
	tray.OnClick(show)
	menu := app.NewMenu()
	menu.Add("显示窗口").OnClick(func(_ *application.Context) { show() })
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(_ *application.Context) { app.Quit() })
	tray.SetMenu(menu)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ready:
			primary.Watch(ctx, func() { application.InvokeAsync(show) })
		case <-ctx.Done():
		}
	}()
	defer func() { cancel(); <-done }()
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}

func writeReport(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(data))
	if err = os.MkdirAll("selfcheck", 0755); err == nil {
		err = os.WriteFile(filepath.Join("selfcheck", "selftest.json"), data, 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
