package config

import (
	"fmt"
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
)

type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

type Serve struct {
	CDPURL              string        `toml:"cdp_url" env:"SERVE_CDP_URL"`
	ExecPath            string        `toml:"exec_path" env:"SERVE_EXEC_PATH"`
	Headless            bool          `toml:"headless" env:"SERVE_HEADLESS" env-default:"true"`
	UserDataDir         string        `toml:"user_data_dir" env:"SERVE_USER_DATA_DIR"`
	Session             string        `toml:"session" env:"SERVE_SESSION"`
	Transport           Transport     `toml:"transport" env:"SERVE_TRANSPORT" env-default:"stdio"`
	HTTPAddr            string        `toml:"http_addr" env:"SERVE_HTTP_ADDR" env-default:":8080"`
	ActionTimeout       time.Duration `toml:"action_timeout" env:"SERVE_ACTION_TIMEOUT" env-default:"30s"`
	StartupTimeout      time.Duration `toml:"startup_timeout" env:"SERVE_STARTUP_TIMEOUT" env-default:"30s"`
	HTTPReadTimeout     time.Duration `toml:"http_read_timeout" env:"SERVE_HTTP_READ_TIMEOUT" env-default:"5s"`
	HTTPShutdownTimeout time.Duration `toml:"http_shutdown_timeout" env:"SERVE_HTTP_SHUTDOWN_TIMEOUT" env-default:"5s"`
	WindowWidth         int           `toml:"window_width" env:"SERVE_WINDOW_WIDTH" env-default:"1280"`
	WindowHeight        int           `toml:"window_height" env:"SERVE_WINDOW_HEIGHT" env-default:"800"`
}

func ParseTransport(value string) (Transport, error) {
	transport := Transport(value)
	switch transport {
	case TransportStdio, TransportHTTP:
		return transport, nil
	default:
		return "", fmt.Errorf("unsupported transport %q", value)
	}
}

func (s Serve) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Transport, validation.Required, validation.In(TransportStdio, TransportHTTP)),
		validation.Field(&s.HTTPAddr, validation.Required),
		validation.Field(&s.ActionTimeout, validation.Required, validation.Min(time.Second)),
		validation.Field(&s.StartupTimeout, validation.Required, validation.Min(time.Second)),
		validation.Field(&s.HTTPReadTimeout, validation.Required, validation.Min(time.Second)),
		validation.Field(&s.HTTPShutdownTimeout, validation.Required, validation.Min(time.Second)),
		validation.Field(&s.WindowWidth, validation.Required, validation.Min(1)),
		validation.Field(&s.WindowHeight, validation.Required, validation.Min(1)),
	)
}
