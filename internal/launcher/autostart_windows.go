package launcher

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	autoStartKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartName = "App Runner"
)

// AutoStartCommand 返回登录时应当执行的命令行：当前可执行文件的完整路径。
func AutoStartCommand() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	quote := string(rune(34))
	return quote + executable + quote, nil
}

// AutoStartEnabled 报告当前可执行文件是否已登记为登录启动项。
func AutoStartEnabled() (bool, error) {
	command, err := AutoStartCommand()
	if err != nil {
		return false, err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.QUERY_VALUE)
	if err != nil {
		return false, nil
	}
	defer key.Close()
	value, _, err := key.GetStringValue(autoStartName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(value), command), nil
}

// SetAutoStart 写入或移除登录启动项。便携版换过位置后需要重新开启一次，
// 这里记录的是当前可执行文件的绝对路径。
func SetAutoStart(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(autoStartName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	command, err := AutoStartCommand()
	if err != nil {
		return err
	}
	return key.SetStringValue(autoStartName, command)
}
