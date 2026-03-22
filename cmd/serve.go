package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/agent"
	agentcursor "github.com/GrayFlash/kirkup-cli/agent/cursor"
	agentgemini "github.com/GrayFlash/kirkup-cli/agent/gemini"
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
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	registry := agent.NewRegistry(
		agentgemini.New(),
		agentcursor.New(),
	)

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
