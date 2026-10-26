package chromium

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
	"github.com/yvv4git/browser-chromium-mcp/internal/ports"
	"go.uber.org/zap"
)

const _engineChromium = "chromium"

var _refPattern = regexp.MustCompile(`^e[0-9]+$`)

type Client struct {
	log               *zap.Logger
	options           Options
	allocCtx          context.Context
	browserCtx        context.Context
	browserCancel     context.CancelFunc
	allocCancel       context.CancelFunc
	selectAllModifier input.Modifier
	mu                sync.Mutex
	sessions          map[string]*session
}

func New(ctx context.Context, options Options, log *zap.Logger) (*Client, error) {
	allocCtx, allocCancel := options.allocatorContext(context.Background())
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	client := &Client{
		log:               log,
		options:           options,
		allocCtx:          allocCtx,
		browserCtx:        browserCtx,
		browserCancel:     browserCancel,
		allocCancel:       allocCancel,
		selectAllModifier: options.selectAllModifier(),
		sessions:          make(map[string]*session),
	}

	if options.CDPURL == "" {
		startCtx, startCancel := context.WithTimeout(context.Background(), options.StartupTimeout)
		defer startCancel()

		if err := chromedp.Run(startCtx); err != nil {
			browserCancel()
			allocCancel()

			return nil, fmt.Errorf("start browser: %w", err)
		}
	} else {
		testCtx, _ := chromedp.NewContext(allocCtx)
		startCtx, startCancel := context.WithTimeout(testCtx, options.StartupTimeout)
		if err := chromedp.Run(startCtx); err != nil {
			startCancel()
			browserCancel()
			allocCancel()

			return nil, fmt.Errorf("start browser: %w", err)
		}
		startCancel()
	}

	log.Info("Browser connected",
		zap.String("cdp_url", options.CDPURL),
		zap.Bool("headless", options.Headless),
	)

	return client, nil
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

	c.browserCancel()
	c.allocCancel()

	return nil
}

func (c *Client) Health(ctx context.Context) (*domain.Health, error) {
	err := c.browserVersion(ctx)

	return &domain.Health{
		OK:               err == nil,
		Engine:           _engineChromium,
		Sessions:         c.sessionCount(),
		BrowserConnected: err == nil,
	}, nil
}

const _pageMetaJS = `(() => { return {url: location.href, title: document.title}; })()`

func (c *Client) runTab(ctx context.Context, t *tab, actions ...chromedp.Action) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.toolCalls++

	runCtx, cancel := context.WithTimeout(ctx, c.options.ActionTimeout)
	defer cancel()

	if err := chromedp.Run(runCtx, actions...); err != nil {
		return fmt.Errorf("run tab action: %w", err)
	}

	return nil
}

func (c *Client) browserVersion(ctx context.Context) error {
	chromedpCtx := chromedp.FromContext(c.browserCtx)
	if chromedpCtx == nil || chromedpCtx.Browser == nil {
		return fmt.Errorf("browser is not initialized")
	}

	if _, _, _, _, _, err := browser.GetVersion().Do(cdp.WithExecutor(ctx, chromedpCtx.Browser)); err != nil {
		return fmt.Errorf("get browser version: %w", err)
	}

	return nil
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
