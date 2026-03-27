package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/agent"
	agentclaude "github.com/GrayFlash/kirkup-cli/agent/claude"
	agentcursor "github.com/GrayFlash/kirkup-cli/agent/cursor"
	agentgemini "github.com/GrayFlash/kirkup-cli/agent/gemini"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List supported agents and their detection status",
	RunE:  runAgents,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
}

func runAgents(_ *cobra.Command, _ []string) error {
	registry := agent.NewRegistry(
		agentgemini.New(),
		agentcursor.New(),
		agentclaude.New(),
	)

	for _, a := range registry.All() {
		status := "not detected"
		if a.Detect() {
			status = "detected"
		}
		fmt.Printf("%-14s  %s\n", a.Name(), status)
		for _, g := range a.WatchGlobs() {
			fmt.Printf("               %s\n", g)
		}
	}
	return nil
}
