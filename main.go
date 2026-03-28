package main

import (
	_ "embed"

	"github.com/GrayFlash/kirkup-cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

//go:embed config/defaults/default.yaml
var defaultConfig []byte

func main() {
	cmd.DefaultConfig = defaultConfig
	cmd.Version = version + " (" + commit + ") " + date
	cmd.Execute()
}
