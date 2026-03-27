//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

// detachProcess configures cmd to start in its own session, detached from
// the calling terminal.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// isRunning returns true if a process with the given PID is alive.
func isRunning(pid int) bool {
	// Signal 0 checks process existence without sending an actual signal.
	err := syscall.Kill(pid, 0)
	return err == nil
}

// stopProcess sends SIGTERM to the process, requesting a graceful shutdown.
func stopProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
