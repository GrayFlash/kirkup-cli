package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the collector daemon in the background",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(_ *cobra.Command, _ []string) error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	// Check if already running.
	if pid, err := readPID(pidPath); err == nil && isRunning(pid) {
		return fmt.Errorf("kirkup is already running (pid %d)", pid)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	logPath, err := logFilePath()
	if err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	cmd := exec.Command(exe, "serve")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	detachProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	if err := writePID(pidPath, cmd.Process.Pid); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}

	fmt.Printf("started (pid %d)\n", cmd.Process.Pid)
	fmt.Printf("logs:    %s\n", logPath)
	return nil
}
