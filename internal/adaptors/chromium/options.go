package chromium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/launcher"
)

const (
	_defaultStartupTimeout = 30 * time.Second
	_defaultActionTimeout  = 30 * time.Second
)

type Options struct {
	CDPURL         string
	ExecPath       string
	Headless       bool
	UserDataDir    string
	WindowWidth    int
	WindowHeight   int
	ActionTimeout  time.Duration
	StartupTimeout time.Duration
}

func (o Options) startupTimeout() time.Duration {
	if o.StartupTimeout <= 0 {
		return _defaultStartupTimeout
	}

	return o.StartupTimeout
}

func (o Options) actionTimeout() time.Duration {
	if o.ActionTimeout <= 0 {
		return _defaultActionTimeout
	}

	return o.ActionTimeout
}

func (o Options) launch(ctx context.Context) (string, *launcher.Launcher, error) {
	if o.CDPURL != "" {
		url, err := resolveControlURL(ctx, o.CDPURL)
		if err != nil {
			return "", nil, err
		}

		return url, nil, nil
	}

	l := launcher.New().Headless(o.Headless)

	if o.WindowWidth > 0 && o.WindowHeight > 0 {
		l.Set("window-size", fmt.Sprintf("%d,%d", o.WindowWidth, o.WindowHeight))
	}

	if o.ExecPath != "" {
		l.Bin(o.ExecPath)
	} else if bin, ok := launcher.LookPath(); ok {
		l.Bin(bin)
	}

	if o.UserDataDir != "" {
		l.UserDataDir(o.UserDataDir)
	}

	launchCtx, cancel := context.WithTimeout(ctx, o.startupTimeout())
	defer cancel()

	url, err := l.Context(launchCtx).Launch()
	if err != nil {
		return "", nil, fmt.Errorf("launch browser: %w", err)
	}

	return url, l, nil
}

func resolveControlURL(ctx context.Context, addr string) (string, error) {
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		return addr, nil
	}

	endpoint := strings.TrimSuffix(addr, "/") + "/json/version"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request %s: %w", endpoint, err)
	}

	req.Close = true

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: %s", endpoint, resp.Status)
	}

	var data struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("parse %s: %w", endpoint, err)
	}
	if data.WebSocketDebuggerURL == "" {
		return "", errors.New("no webSocketDebuggerUrl in " + endpoint)
	}

	return data.WebSocketDebuggerURL, nil
}
