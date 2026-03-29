package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GrayFlash/kirkup-cli/config"
)

func generateDashboardCompose(cfg *config.Config, composePath string) error {
	if cfg.Store.Driver == "postgres" {
		return fmt.Errorf("web dashboards currently only support SQLite")
	}

	var template string

	switch cfg.Retro.Dashboard {
	case "datasette":
		template = `services:
  datasette:
    image: datasetteproject/datasette:latest
    container_name: kirkup-datasette
    ports:
      - "{port}:{port}"
    volumes:
      - {sqlite_path}:/data/kirkup.db:ro
    command: ["datasette", "-h", "0.0.0.0", "-p", "{port}", "/data/kirkup.db"]
    restart: unless-stopped`
	case "lite-queen":
		template = `services:
  lite-queen:
    image: ghcr.io/kivs/lite-queen:latest
    container_name: kirkup-lite-queen
    ports:
      - "{port}:8080"
    volumes:
      - {sqlite_path}:/data/kirkup.db:ro
    restart: unless-stopped`
	case "none", "":
		return nil
	default:
		return fmt.Errorf("unsupported dashboard: %s", cfg.Retro.Dashboard)
	}

	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	sqlitePath := cfg.Store.SQLite.Path
	if !filepath.IsAbs(sqlitePath) {
		sqlitePath = filepath.Join(dir, sqlitePath)
	}

	port := strconv.Itoa(cfg.Retro.Port)
	if port == "0" {
		port = "8001"
	}

	content := strings.ReplaceAll(template, "{sqlite_path}", sqlitePath)
	content = strings.ReplaceAll(content, "{port}", port)

	if err := os.MkdirAll(filepath.Dir(composePath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(composePath, []byte(content), 0o600)
}
