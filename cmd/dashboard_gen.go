package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GrayFlash/kirkup-cli/config"
)

func generateDashboardCompose(cfg *config.Config, composePath string) error {
	var template string

	switch cfg.Retro.Dashboard {
	case "datasette":
		template = "version: '3.7'\nservices:\n  datasette:\n    image: datasetteproject/datasette:latest\n    container_name: kirkup-datasette\n    ports:\n      - '8001:8001'\n    volumes:\n      - {sqlite_path}:/data/kirkup.db:ro\n    command: [\"datasette\", \"-h\", \"0.0.0.0\", \"-p\", \"8001\", \"/data/kirkup.db\"]\n    restart: unless-stopped"
	case "lite-queen":
		template = "version: '3.7'\nservices:\n  lite-queen:\n    image: kivsegransen/lite-queen:latest\n    container_name: kirkup-lite-queen\n    ports:\n      - '8001:8001'\n    volumes:\n      - {sqlite_path}:/data/kirkup.db:ro\n    restart: unless-stopped"
	default:
		return nil
	}

	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	sqlitePath := cfg.Store.SQLite.Path
	if !filepath.IsAbs(sqlitePath) {
		sqlitePath = filepath.Join(dir, sqlitePath)
	}

	content := strings.ReplaceAll(template, "{sqlite_path}", sqlitePath)

	if err := os.MkdirAll(filepath.Dir(composePath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(composePath, []byte(content), 0o600)
}
