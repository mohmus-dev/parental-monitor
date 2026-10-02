//go:build !windows

package monitor

import (
    "os/exec"
    "strings"
)

type WindowMonitor struct {
    lastTitle string
}

func NewWindowMonitor() *WindowMonitor {
    return &WindowMonitor{}
}

func (w *WindowMonitor) GetActiveWindowTitle() string {
    // Linux: use xdotool or wmctrl (requires X11)
    cmd := exec.Command("xdotool", "getactivewindow", "getwindowname")
    out, err := cmd.Output()
    if err != nil {
        // Fallback to wmctrl
        cmd = exec.Command("wmctrl", "-a", ":ACTIVE:")
        out, _ = cmd.Output()
    }

    title := strings.TrimSpace(string(out))
    if title != "" {
        w.lastTitle = title
    }
    return title
}