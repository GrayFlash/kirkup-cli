package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the local analytics dashboard (requires Docker)",
	Long: `Launch a local Metabase dashboard instance using Docker Compose.
The dashboard maps to your local kirkup database for rich visualisations.`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(_ *cobra.Command, _ []string) error {
	// Check if docker-compose.yaml exists in the expected location
	// For now, assume it's in the repo or a known install location.
	// Since this is a CLI, we might want to embed the yaml or write it to ~/.kirkup/dashboard/
	
	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	dashDir := dir + "/dashboard"
	if err := os.MkdirAll(dashDir, 0o755); err != nil {
		return err
	}
	
	composePath := dashDir + "/docker-compose.yaml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		fmt.Println("creating dashboard configuration...")
		if err := os.WriteFile(composePath, []byte(dashboardCompose), 0o644); err != nil {
			return err
		}
	}

	fmt.Println("launching dashboard via docker-compose...")
	cmd := exec.Command("docker-compose", "-f", composePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start dashboard: %w (is Docker installed and running?)", err)
	}

	fmt.Println("\ndashboard is starting up!")
	fmt.Println("access it at: http://localhost:3000")
	fmt.Println("\nnote: the first time you run this, it may take a minute to pull the image.")
	return nil
}

const dashboardCompose = `version: "3.7"

services:
  metabase:
    image: metabase/metabase:latest
    container_name: kirkup-dashboard
    ports:
      - "3000:3000"
    volumes:
      - ./metabase-data:/metabase-data
      - ../kirkup.db:/data/kirkup.db:ro
    environment:
      - MB_DB_FILE=/metabase-data/metabase.db
    restart: unless-stopped
`
