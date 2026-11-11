//go:build !windows

package launcher

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
)

func OpenDirectory(path string) error {
	program := "xdg-open"
	if runtime.GOOS == "darwin" {
		program = "open"
	}
	cmd := exec.Command(program, path)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
func EditProfile(path string) error { return OpenDirectory(path) }
func Launch(a App) error {
	if a.RunAsAdmin {
		return errors.New("管理员启动目前仅支持 Windows")
	}
	entry, err := ResolveEntry(a, a.Entry)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	if a.Type == "CLI" {
		cmd = exec.Command("x-terminal-emulator", "-e", "sh", "-c", `"$1"; exec "${SHELL:-/bin/sh}"`, "launcher", entry)
	} else if a.Type == "GUI" {
		cmd = exec.Command(entry)
	} else {
		return errors.New("无效的应用类型")
	}
	cmd.Dir = filepath.Dir(entry)
	if err = cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
func quietCommand(cmd *exec.Cmd)      {}
func fileVersion(path string) string  { return "" }
func ShowStartupError(message string) {}
