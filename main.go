package main

import (
	"os"

	"github.com/yvv4git/browser-chromium-mcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
