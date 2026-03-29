package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

type composeService struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name"`
	Ports         []string          `yaml:"ports"`
	Volumes       []string          `yaml:"volumes"`
	Environment   map[string]string `yaml:"environment"`
	Restart       string            `yaml:"restart"`
}

type composeRoot struct {
	Version  string                    `yaml:"version"`
	Services map[string]composeService `yaml:"services"`
}

func runDashboard(_ *cobra.Command, _ []string) error {
	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	dashDir := filepath.Join(dir, "dashboard")
	if err := os.MkdirAll(dashDir, 0o700); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	composePath := filepath.Join(dashDir, "docker-compose.yaml")

	root := composeRoot{
		Version: "3.7",
		Services: map[string]composeService{
			"metabase": {
				Image:         "metabase/metabase:latest",
				ContainerName: "kirkup-dashboard",
				Ports:         []string{"3000:3000"},
				Volumes:       []string{"./metabase-data:/metabase-data"},
				Environment: map[string]string{
					"MB_DB_FILE": "/metabase-data/metabase.db",
				},
				Restart: "unless-stopped",
			},
		},
	}

	svc := root.Services["metabase"]

	switch cfg.Store.Driver {
	case "sqlite":
		sqlitePath := cfg.Store.SQLite.Path
		if !filepath.IsAbs(sqlitePath) {
			sqlitePath = filepath.Join(dir, sqlitePath)
		}
		svc.Volumes = append(svc.Volumes, fmt.Sprintf("%s:/data/kirkup.db:ro", sqlitePath))
		svc.Environment["MB_DB_TYPE"] = "sqlite"
		svc.Environment["MB_DB_DBNAME"] = "/data/kirkup.db"
	case "postgres":
		svc.Environment["MB_DB_TYPE"] = "postgres"
		svc.Environment["MB_DB_CONNECTION_URI"] = cfg.Store.PG.DSN
	default:
		return fmt.Errorf("unsupported store driver for dashboard: %q", cfg.Store.Driver)
	}

	root.Services["metabase"] = svc

	composeContent, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshal compose config: %w", err)
	}

	fmt.Println("launching dashboard via docker-compose...")
	if err := os.WriteFile(composePath, composeContent, 0o600); err != nil {
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
	fmt.Println("note: Metabase is a JVM application and requires ~500MB-1GB of RAM to run smoothly.")
	return nil
}
