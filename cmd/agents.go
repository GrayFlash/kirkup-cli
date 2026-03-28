package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

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
	registry := newAgentRegistry()

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
