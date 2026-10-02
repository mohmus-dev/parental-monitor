//go:build windows

package monitor

import (
	"unicode/utf16"
	"unsafe"
)

var (
	procGetForegroundWindow2     = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

type WindowMonitor struct {
	lastTitle string
}

func NewWindowMonitor() *WindowMonitor {
	return &WindowMonitor{}
}

func (w *WindowMonitor) GetActiveWindowTitle() string {
	hwnd, _, _ := procGetForegroundWindow2.Call()
	if hwnd == 0 {
		return ""
	}

	// Get window text length
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return w.lastTitle
	}

	// Allocate buffer for title
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))

	title := string(utf16.Decode(buf))
	if title != "" {
		w.lastTitle = title
	}
	return title
}
