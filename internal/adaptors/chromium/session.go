package chromium

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
	"go.uber.org/zap"
)

type pageMeta struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type session struct {
	key    string
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	tabs   map[string]*tab
}

type tab struct {
	id        string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	url       string
	title     string
	visited   []string
	toolCalls int
	refsCount int
}

func (c *Client) newSession(ctx context.Context, key string) (*session, error) {
	sessCtx, cancel := chromedp.NewContext(c.browserCtx, chromedp.WithNewBrowserContext())

	startCtx, startCancel := context.WithTimeout(ctx, c.options.ActionTimeout)
	defer startCancel()

	if err := chromedp.Run(startCtx); err != nil {
		cancel()

		return nil, fmt.Errorf("create browser context: %w", err)
	}

	chromedpCtx := chromedp.FromContext(sessCtx)
	sess := &session{key: key, ctx: sessCtx, cancel: cancel, tabs: make(map[string]*tab)}

	if chromedpCtx.Target != nil {
		if err := c.closeTarget(startCtx, chromedpCtx.Target.TargetID); err != nil {
			cancel()

			return nil, fmt.Errorf("close session bootstrap target: %w", err)
		}
	}

	c.log.Debug("Session created", zap.String("session", key))

	return sess, nil
}

func (c *Client) closeTarget(ctx context.Context, id target.ID) error {
	chromedpCtx := chromedp.FromContext(c.browserCtx)
	if chromedpCtx == nil || chromedpCtx.Browser == nil {
		return fmt.Errorf("browser is not initialized")
	}

	if err := target.CloseTarget(id).Do(cdp.WithExecutor(ctx, chromedpCtx.Browser)); err != nil {
		return fmt.Errorf("close target %s: %w", id, err)
	}

	return nil
}

func (c *Client) openTab(ctx context.Context, sess *session, url string) (*domain.Tab, error) {
	tabCtx, cancel := chromedp.NewContext(sess.ctx)
	t := &tab{ctx: tabCtx, cancel: cancel}

	var meta pageMeta
	if err := c.runTab(ctx, t, chromedp.Navigate(url), chromedp.Evaluate(_pageMetaJS, &meta)); err != nil {
		cancel()

		return nil, fmt.Errorf("open tab: %w", err)
	}

	chromedpCtx := chromedp.FromContext(tabCtx)
	t.id = string(chromedpCtx.Target.TargetID)
	t.url = meta.URL
	t.title = meta.Title
	t.visited = append(t.visited, meta.URL)

	if err := sess.addTab(t); err != nil {
		cancel()

		return nil, err
	}

	c.log.Debug("Tab opened",
		zap.String("session", sess.key),
		zap.String("tab", t.id),
		zap.String("url", url),
	)

	return &domain.Tab{TabID: t.id, TargetID: t.id, URL: t.url, Title: t.title}, nil
}

func (c *Client) updateMeta(t *tab, meta pageMeta) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if meta.URL != "" && meta.URL != t.url {
		t.visited = append(t.visited, meta.URL)
		t.url = meta.URL
	}

	if meta.Title != "" {
		t.title = meta.Title
	}
}

func (c *Client) OpenTab(ctx context.Context, sessionKey, url string) (*domain.Tab, error) {
	sess, err := c.ensureSession(ctx, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}

	tab, err := c.openTab(ctx, sess, url)
	if err != nil {
		return nil, err
	}

	return tab, nil
}

func (c *Client) CloseTab(_ context.Context, sessionKey, tabID string) error {
	sess, err := c.getSession(sessionKey)
	if err != nil {
		return err
	}

	t, err := sess.removeTab(tabID)
	if err != nil {
		return err
	}

	t.close()

	c.log.Debug("Tab closed", zap.String("session", sessionKey), zap.String("tab", tabID))

	return nil
}

func (c *Client) CloseSession(_ context.Context, sessionKey string) error {
	sess := c.removeSession(sessionKey)
	if sess == nil {
		return domain.ErrSessionNotFound
	}

	sess.close()

	c.log.Debug("Session closed", zap.String("session", sessionKey))

	return nil
}

func (c *Client) ListTabs(_ context.Context, sessionKey string) (*domain.ListTabs, error) {
	sess, err := c.getSession(sessionKey)
	if err != nil {
		return nil, err
	}

	return &domain.ListTabs{Tabs: sess.listTabs()}, nil
}

func (s *session) addTab(t *tab) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tabs[t.id]; ok {
		return fmt.Errorf("tab %s already exists", t.id)
	}

	s.tabs[t.id] = t

	return nil
}

func (s *session) removeTab(tabID string) (*tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tabs[tabID]
	if !ok {
		return nil, domain.ErrTabNotFound
	}

	delete(s.tabs, tabID)

	return t, nil
}

func (s *session) tab(tabID string) (*tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tabs[tabID]
	if !ok {
		return nil, domain.ErrTabNotFound
	}

	return t, nil
}

func (s *session) listTabs() []domain.TabInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	tabs := make([]domain.TabInfo, 0, len(s.tabs))
	for _, t := range s.tabs {
		t.mu.Lock()
		tabs = append(tabs, domain.TabInfo{TabID: t.id, URL: t.url, Title: t.title})
		t.mu.Unlock()
	}

	return tabs
}

func (s *session) close() {
	s.mu.Lock()
	tabs := make([]*tab, 0, len(s.tabs))
	for _, t := range s.tabs {
		tabs = append(tabs, t)
	}
	s.tabs = nil
	s.mu.Unlock()

	for _, t := range tabs {
		t.close()
	}

	s.cancel()
}

func (t *tab) close() {
	t.cancel()
}
