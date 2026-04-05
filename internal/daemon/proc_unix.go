//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// SysProcAttr returns platform-specific process attributes for Unix.
// Setsid creates a new session, detaching from the terminal.
func SysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// StopProcess sends SIGTERM and waits up to 10 seconds, then SIGKILL.
func StopProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to %d: %w", pid, err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", pid, err)
		}
		return nil
	}
}

// IsProcessRunning checks if a process with the given PID is alive.
func IsProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
