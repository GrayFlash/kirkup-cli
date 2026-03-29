package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the collector daemon",
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(_ *cobra.Command, _ []string) error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	pid, err := readPID(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("kirkup is not running")
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}

	if !isRunning(pid) {
		_ = os.Remove(pidPath)
		fmt.Println("kirkup is not running (stale pid file removed)")
		return nil
	}

	if err := stopProcess(pid); err != nil {
		return fmt.Errorf("stop process: %w", err)
	}

	_ = os.Remove(pidPath)
	fmt.Printf("stopped (pid %d)\n", pid)
	return nil
}
