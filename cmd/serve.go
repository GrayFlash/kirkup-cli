package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/collector"
)

var serveCmd = &cobra.Command{
	Use:    "serve",
	Short:  "Run the collector in the foreground (used internally by start)",
	Hidden: true,
	RunE:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) error {
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	registry := newAgentRegistry()

	level := slog.LevelInfo
	if cfg.Daemon.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	c := collector.New(registry, s, cfg, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return c.Start(ctx)
}
