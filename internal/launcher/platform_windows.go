package launcher

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func shellOpen(verb, file, args, cwd string) error {
	v, e := windows.UTF16PtrFromString(verb)
	if e != nil {
		return e
	}
	f, e := windows.UTF16PtrFromString(file)
	if e != nil {
		return e
	}
	p, e := windows.UTF16PtrFromString(args)
	if e != nil {
		return e
	}
	d, e := windows.UTF16PtrFromString(cwd)
	if e != nil {
		return e
	}
	return windows.ShellExecute(0, v, f, p, d, windows.SW_SHOWNORMAL)
}

func OpenDirectory(path string) error { return shellOpen("open", path, "", path) }
func EditProfile(path string) error {
	return shellOpen("open", "notepad.exe", syscall.EscapeArg(path), filepath.Dir(path))
}

func psQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
func encodedCommand(script string) string {
	units := utf16.Encode([]rune(script))
	bytes := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(bytes[i*2:], u)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func Launch(a App) error {
	if a.Type != "GUI" && a.Type != "CLI" {
		return fmt.Errorf("无效的应用类型：%s", a.Type)
	}
	entry, err := ResolveEntry(a, a.Entry)
	if err != nil {
		return err
	}
	file, args := entry, ""
	cwd := filepath.Dir(entry)
	if a.Type == "CLI" {
		file, err = exec.LookPath("pwsh.exe")
		if err != nil {
			file, err = exec.LookPath("powershell.exe")
		}
		if err != nil {
			return fmt.Errorf("找不到 PowerShell：%w", err)
		}
		script := "$env:Path = " + psQuote(cwd+";") + " + $env:Path; & " + psQuote(entry)
		args = "-NoExit -EncodedCommand " + encodedCommand(script)
	}
	verb := "open"
	if a.RunAsAdmin {
		verb = "runas"
	}
	err = shellOpen(verb, file, args, cwd)
	if err == windows.ERROR_ELEVATION_REQUIRED && verb != "runas" {
		err = shellOpen("runas", file, args, cwd)
	}
	if err != nil {
		return fmt.Errorf("无法运行 %s：%w", a.Title, err)
	}
	return nil
}

func quietCommand(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }

func ShowStartupError(message string) {
	text, _ := windows.UTF16PtrFromString(message)
	title, _ := windows.UTF16PtrFromString("App Runner - 无法启动")
	windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}

func fileVersion(path string) string {
	var handle windows.Handle
	size, err := windows.GetFileVersionInfoSize(path, &handle)
	if err != nil || size == 0 {
		return ""
	}
	buffer := make([]byte, size)
	if windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buffer[0])) != nil {
		return ""
	}
	var info *windows.VS_FIXEDFILEINFO
	var length uint32
	if windows.VerQueryValue(unsafe.Pointer(&buffer[0]), `\`, unsafe.Pointer(&info), &length) != nil || info == nil || length < uint32(unsafe.Sizeof(*info)) {
		return ""
	}
	parts := []uint32{info.FileVersionMS >> 16, info.FileVersionMS & 0xffff, info.FileVersionLS >> 16, info.FileVersionLS & 0xffff}
	for len(parts) > 1 && parts[len(parts)-1] == 0 {
		parts = parts[:len(parts)-1]
	}
	result := make([]string, len(parts))
	for i, n := range parts {
		result[i] = fmt.Sprint(n)
	}
	return strings.Join(result, ".")
}
