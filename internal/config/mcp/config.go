package mcp

import (
	"fmt"

	"github.com/yvv4git/browser-chromium-mcp/internal/config"
)

type Config struct {
	Log   config.Log   `toml:"log"`
	Serve config.Serve `toml:"serve"`
}

func Load(path string, cfg *Config) error {
	return config.Load(path, cfg)
}

func LoadEnv(cfg *Config) error {
	return config.LoadEnv(cfg)
}

func (c Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log: %w", err)
	}

	if err := c.Serve.Validate(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
