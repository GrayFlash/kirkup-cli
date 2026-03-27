package main

import (
	_ "embed"

	"github.com/GrayFlash/kirkup-cli/cmd"
)

//go:embed configs/default.yaml
var defaultConfig []byte

func main() {
	cmd.DefaultConfig = defaultConfig
	cmd.Execute()
}
