//go:build !windows

package launcher

import "errors"

func AutoStartEnabled() (bool, error) { return false, nil }

func SetAutoStart(bool) error { return errors.New("当前平台不支持开机自启") }
