//go:build !windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// daemonize re-launches the program as a detached background process (Linux/macOS)
func daemonize(configPath string) {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("Failed to get config path: %v", err)
	}

	cmd := exec.Command(exe, "-config", absConfig)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // create a new session (fully detached, no controlling terminal)
	}

	// Redirect output to log file
	logFile, err := os.OpenFile("daemon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	// Save PID for later stopping
	pidFile := "monitor.pid"
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644)

	// Release the process handle so it keeps running after we exit
	cmd.Process.Release()

	fmt.Println("✅ Daemon started in background (PID:", cmd.Process.Pid, ")")
	fmt.Println("   Log file: daemon.log")
	fmt.Println("   To stop: parental-monitor -stop")
}

// stopDaemon kills the running background process (Linux/macOS)
func stopDaemon() {
	pidData, err := os.ReadFile("monitor.pid")
	if err != nil {
		log.Fatalf("No PID file found. Is the daemon running? (%v)", err)
	}

	var pid int
	fmt.Sscanf(string(pidData), "%d", &pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("Failed to find process: %v", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Fatalf("Failed to signal process: %v", err)
	}

	os.Remove("monitor.pid")
	fmt.Println("🛑 Daemon stopped.")
}
