package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

func LoadDotEnv() error {
	_, err := os.Stat(".env")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("stat .env: %w", err)
	}

	if err := godotenv.Load(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

func Load[T any](path string, cfg *T) error {
	if err := LoadDotEnv(); err != nil {
		return err
	}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

func LoadEnv[T any](cfg *T) error {
	if err := LoadDotEnv(); err != nil {
		return err
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return fmt.Errorf("read env: %w", err)
	}

	return nil
}
