package chromium

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
	"github.com/yvv4git/browser-chromium-mcp/internal/ports"
	"go.uber.org/zap"
)

const _engineChromium = "chromium"

var _refPattern = regexp.MustCompile(`^e[0-9]+$`)

type Client struct {
	log      *zap.Logger
	options  Options
	browser  *rod.Browser
	launcher *launcher.Launcher
	mu       sync.Mutex
	sessions map[string]*session
}

func New(ctx context.Context, options Options, log *zap.Logger) (*Client, error) {
	controlURL, l, err := options.launch(ctx)
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(controlURL).NoDefaultDevice()

	if err := connectBrowser(ctx, browser, options.startupTimeout()); err != nil {
		if l != nil {
			l.Kill()
		}

		return nil, fmt.Errorf("connect browser: %w", err)
	}

	log.Info("Browser connected",
		zap.String("cdp_url", options.CDPURL),
		zap.Bool("headless", options.Headless),
	)

	return &Client{
		log:      log,
		options:  options,
		browser:  browser,
		launcher: l,
		sessions: make(map[string]*session),
	}, nil
}

func connectBrowser(ctx context.Context, browser *rod.Browser, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- browser.Connect()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("connect timed out after %s", timeout)
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	sessions := make([]*session, 0, len(c.sessions))
	for _, sess := range c.sessions {
		sessions = append(sessions, sess)
	}
	c.sessions = nil
	c.mu.Unlock()

	for _, sess := range sessions {
		sess.close()
	}

	if c.browser != nil {
		_ = c.browser.Close()
	}

	if c.launcher != nil {
		c.launcher.Kill()
	}

	return nil
}

func (c *Client) Health(ctx context.Context) (*domain.Health, error) {
	_, err := c.browser.Context(ctx).Version()

	return &domain.Health{
		OK:               err == nil,
		Engine:           _engineChromium,
		Sessions:         c.sessionCount(),
		BrowserConnected: err == nil,
	}, nil
}

func (c *Client) sessionCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.sessions)
}

func (c *Client) getSession(key string) (*session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sess, ok := c.sessions[key]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}

	return sess, nil
}

func (c *Client) lookupSession(key string) (*session, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sess, ok := c.sessions[key]

	return sess, ok
}

func (c *Client) ensureSession(ctx context.Context, key string) (*session, error) {
	if sess, ok := c.lookupSession(key); ok {
		return sess, nil
	}

	sess, err := c.newSession(ctx, key)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.sessions[key]; ok {
		sess.close()

		return existing, nil
	}

	c.sessions[key] = sess

	return sess, nil
}

func (c *Client) removeSession(key string) *session {
	c.mu.Lock()
	defer c.mu.Unlock()

	sess, ok := c.sessions[key]
	if !ok {
		return nil
	}

	delete(c.sessions, key)

	return sess
}

func (c *Client) tabFor(sessionKey, tabID string) (*session, *tab, error) {
	sess, err := c.getSession(sessionKey)
	if err != nil {
		return nil, nil, err
	}

	t, err := sess.tab(tabID)
	if err != nil {
		return nil, nil, err
	}

	return sess, t, nil
}

func (c *Client) elementSelector(ref, selector string) (string, error) {
	if ref != "" {
		return c.refSelector(ref)
	}

	if selector != "" {
		return selector, nil
	}

	return "", domain.ErrEmptySelector
}

func (c *Client) refSelector(ref string) (string, error) {
	if !_refPattern.MatchString(ref) {
		return "", fmt.Errorf("%w: %q", domain.ErrInvalidRef, ref)
	}

	return `[data-mcp-ref="` + ref + `"]`, nil
}

var _ ports.BrowserClient = (*Client)(nil)
