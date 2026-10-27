package chromium

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
	"github.com/yvv4git/browser-chromium-mcp/internal/ports"
	"go.uber.org/zap"
)

const _engineChromium = "chromium"

var _refPattern = regexp.MustCompile(`^e[0-9]+$`)

type Client struct {
	log        *zap.Logger
	options    Options
	mu         sync.Mutex
	wsURL      string
	browserID  string
	generation uint64
	launcher   *launcher.Launcher
	sessions   map[string]*session
}

type conn struct {
	browser *rod.Browser
	ws      *cdp.WebSocket
	gen     uint64
}

func (cn *conn) close() {
	_ = cn.ws.Close()
}

func New(ctx context.Context, options Options, log *zap.Logger) (*Client, error) {
	c := &Client{
		log:      log,
		options:  options,
		sessions: make(map[string]*session),
	}

	if options.CDPURL == "" {
		wsURL, l, err := options.launch(ctx)
		if err != nil {
			return nil, err
		}

		c.wsURL = wsURL
		c.launcher = l
	}

	log.Info("Browser client initialized",
		zap.String("cdp_url", options.CDPURL),
		zap.Bool("headless", options.Headless),
		zap.String("mode", "on-demand"),
	)

	return c, nil
}

func (c *Client) controlURL(ctx context.Context) (string, error) {
	if c.options.CDPURL != "" {
		return resolveControlURL(ctx, c.options.CDPURL)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wsURL == "" {
		return "", errors.New("browser is not launched")
	}

	return c.wsURL, nil
}

func (c *Client) open(ctx context.Context) (*conn, error) {
	wsURL, err := c.controlURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve cdp url: %w", err)
	}

	gen := c.generation
	c.mu.Lock()
	if c.browserID == "" {
		c.generation = 1
	} else if c.browserID != wsURL {
		c.generation++
	}
	c.browserID = wsURL
	gen = c.generation

	for key, sess := range c.sessions {
		if sess.gen != gen {
			delete(c.sessions, key)
		}
	}
	c.mu.Unlock()

	connectCtx, cancel := context.WithTimeout(ctx, c.options.startupTimeout())
	defer cancel()

	ws := &cdp.WebSocket{}
	if err := ws.Connect(connectCtx, wsURL, nil); err != nil {
		return nil, fmt.Errorf("connect cdp: %w", err)
	}

	browser := rod.New().Client(cdp.New().Start(ws)).NoDefaultDevice()

	if err := connectBrowser(connectCtx, browser, c.options.startupTimeout()); err != nil {
		_ = ws.Close()

		return nil, fmt.Errorf("connect browser: %w", err)
	}

	return &conn{browser: browser, ws: ws, gen: gen}, nil
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
	l := c.launcher
	c.sessions = nil
	c.launcher = nil
	c.mu.Unlock()

	if l != nil {
		l.Kill()
	}

	return nil
}

func (c *Client) Health(ctx context.Context) (*domain.Health, error) {
	connected := false

	cn, err := c.open(ctx)
	if err == nil {
		pingCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
		_, err = cn.browser.Context(pingCtx).Version()
		cancel()
		cn.close()

		connected = err == nil
	}

	return &domain.Health{
		OK:               connected,
		Engine:           _engineChromium,
		Sessions:         c.sessionCount(),
		BrowserConnected: connected,
	}, nil
}

func (c *Client) sessionCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.sessions)
}

func normalizeSessionKey(key string) string {
	if key == "" {
		return _defaultSessionKey
	}

	return key
}

func (c *Client) getSession(key string) (*session, error) {
	key = normalizeSessionKey(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	sess, ok := c.sessions[key]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}

	return sess, nil
}

func (c *Client) ensureSession(ctx context.Context, cn *conn, key string) (*session, error) {
	key = normalizeSessionKey(key)

	c.mu.Lock()
	sess, ok := c.sessions[key]
	c.mu.Unlock()

	if ok && sess.gen == cn.gen {
		return sess, nil
	}

	sess, err := c.newSession(ctx, cn, key)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.sessions[key]; ok && existing.gen == cn.gen {
		return existing, nil
	}

	c.sessions[key] = sess

	return sess, nil
}

func (c *Client) removeSession(key string) *session {
	key = normalizeSessionKey(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	sess, ok := c.sessions[key]
	if !ok {
		return nil
	}

	delete(c.sessions, key)

	return sess
}

func (c *Client) tabFor(cn *conn, sessionKey, tabID string) (*session, *tab, error) {
	sess, err := c.getSession(sessionKey)
	if err != nil {
		return nil, nil, err
	}

	if sess.gen != cn.gen {
		return nil, nil, domain.ErrSessionNotFound
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
