package cmd

import (

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open the config file in $EDITOR",
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(_ *cobra.Command, _ []string) error {
	cfgPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	return openInEditor(cfgPath)
}
