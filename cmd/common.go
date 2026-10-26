package cmd

import (
	"github.com/yvv4git/browser-chromium-mcp/internal/config"
)

func configParseTransport(v string) (config.Transport, error) {
	return config.ParseTransport(v)
}
