package config

import (
	"github.com/go-ozzo/ozzo-validation/v4"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

type Log struct {
	Level string `toml:"level" env:"LOG_LEVEL" env-default:"info"`
}

func (l Log) Validate() error {
	return validation.Validate(&l.Level, validation.Required, validation.In(LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError))
}
