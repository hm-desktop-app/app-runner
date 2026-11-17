//go:build !windows

package launcher

import (
	"errors"
	"os"
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
func EditProfile(path string) error          { return OpenDirectory(path) }
func Launch(a App) error {
	if a.RunAsAdmin {
		return errors.New("管理员启动目前仅支持 Windows")
	}
	entry, err := ResolveEntry(a, a.Entry)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch a.Type {
	case "CLI":
		// 命令行应用不执行入口，只打开注入了入口目录的 shell，工作目录取用户目录。
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return homeErr
		}
		cmd = exec.Command("x-terminal-emulator", "-e", "sh", "-c", `export PATH="$1:$PATH"; exec "${SHELL:-/bin/sh}"`, "launcher", filepath.Dir(entry))
		cmd.Dir = home
	case "GUI":
		cmd = exec.Command(entry)
		cmd.Dir = filepath.Dir(entry)
	default:
		return errors.New("无效的应用类型")
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
func quietCommand(cmd *exec.Cmd)            {}
func fileVersion(path string) string        { return "" }
func finalPath(path string) (string, error) { return filepath.EvalSymlinks(path) }
func ShowStartupError(message string)       {}
