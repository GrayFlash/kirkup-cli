package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	dashDir := filepath.Join(dir, "dashboard")
	if err := os.MkdirAll(dashDir, 0o755); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	composePath := filepath.Join(dashDir, "docker-compose.yaml")
	var composeContent string

	if cfg.Store.Driver == "sqlite" {
		sqlitePath := cfg.Store.SQLite.Path
		if !filepath.IsAbs(sqlitePath) {
			sqlitePath = filepath.Join(dir, sqlitePath)
		}
		composeContent = fmt.Sprintf(dashboardComposeSQLite, sqlitePath)
	} else if cfg.Store.Driver == "postgres" {
		composeContent = fmt.Sprintf(dashboardComposePostgres, cfg.Store.PG.DSN)
	} else {
		return fmt.Errorf("unsupported store driver for dashboard: %q", cfg.Store.Driver)
	}

	fmt.Println("launching dashboard via docker-compose...")
	if err := os.WriteFile(composePath, []byte(composeContent), 0o644); err != nil {
		return err
	}

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

const dashboardComposeSQLite = `version: "3.7"

services:
  metabase:
    image: metabase/metabase:latest
    container_name: kirkup-dashboard
    ports:
      - "3000:3000"
    volumes:
      - ./metabase-data:/metabase-data
      - %s:/data/kirkup.db:ro
    environment:
      - MB_DB_FILE=/metabase-data/metabase.db
      - MB_DB_TYPE=sqlite
      - MB_DB_DBNAME=/data/kirkup.db
    restart: unless-stopped
`

const dashboardComposePostgres = `version: "3.7"

services:
  metabase:
    image: metabase/metabase:latest
    container_name: kirkup-dashboard
    ports:
      - "3000:3000"
    volumes:
      - ./metabase-data:/metabase-data
    environment:
      - MB_DB_FILE=/metabase-data/metabase.db
      - MB_DB_TYPE=postgres
      - MB_DB_CONNECTION_URI=%s
    restart: unless-stopped
`
