//go:build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

const detachedProcess = 0x00000008

// SysProcAttr returns platform-specific process attributes for Windows.
// DETACHED_PROCESS ensures the child has no console window.
func SysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: detachedProcess,
	}
}

// StopProcess kills the process immediately on Windows.
func StopProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID is alive on Windows.
func IsProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess always succeeds; try to open the process handle
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}
