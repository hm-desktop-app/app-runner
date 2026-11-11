package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"unsafe"
)

var dwmSetWindowAttribute = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")

func applyNativeTheme(window *application.WebviewWindow, dark bool) {
	handle := window.NativeWindow()
	if handle == nil {
		return
	}
	value := int32(0)
	if dark {
		value = 1
	}
	dwmSetWindowAttribute.Call(uintptr(handle), 20, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}
