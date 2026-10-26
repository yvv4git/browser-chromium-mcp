package chromium

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
	"go.uber.org/zap"
)

type pageMeta struct {
	URL   string
	Title string
}

type session struct {
	key     string
	browser *rod.Browser
	mu      sync.Mutex
	tabs    map[string]*tab
}

type tab struct {
	id        string
	page      *rod.Page
	mu        sync.Mutex
	url       string
	title     string
	visited   []string
	toolCalls int
	refsCount int
}

func (c *Client) newSession(_ context.Context, key string) (*session, error) {
	incognito, err := c.browser.Incognito()
	if err != nil {
		return nil, fmt.Errorf("create browser context: %w", err)
	}

	sess := &session{
		key:     key,
		browser: incognito,
		tabs:    make(map[string]*tab),
	}

	c.log.Debug("Session created", zap.String("session", key))

	return sess, nil
}

func (c *Client) actionPage(ctx context.Context, t *tab) (*rod.Page, context.CancelFunc) {
	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())

	return t.page.Context(runCtx), cancel
}

func (c *Client) pageMeta(page *rod.Page) pageMeta {
	info, err := page.Info()
	if err != nil {
		return pageMeta{}
	}

	return pageMeta{URL: info.URL, Title: info.Title}
}

func (c *Client) openTab(ctx context.Context, sess *session, url string) (*domain.Tab, error) {
	page, err := sess.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("open tab: %w", err)
	}

	t := &tab{id: string(page.TargetID), page: page}

	if err := c.navigateTab(ctx, t, url); err != nil {
		t.close()

		return nil, fmt.Errorf("open tab: %w", err)
	}

	meta := c.readMeta(t)

	if err := sess.addTab(t); err != nil {
		t.close()

		return nil, err
	}

	c.log.Debug("Tab opened",
		zap.String("session", sess.key),
		zap.String("tab", t.id),
		zap.String("url", url),
	)

	return &domain.Tab{TabID: t.id, TargetID: t.id, URL: meta.URL, Title: meta.Title}, nil
}

func (c *Client) runTab(ctx context.Context, t *tab, action func(page *rod.Page) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.toolCalls++

	page, cancel := c.actionPage(ctx, t)
	defer cancel()

	if err := action(page); err != nil {
		return err
	}

	c.applyMeta(t, c.pageMeta(page))

	return nil
}

func (c *Client) waitLoad(page *rod.Page) {
	if err := page.WaitLoad(); err != nil {
		c.log.Debug("Page load event did not fire", zap.Error(err))
	}
}

func (c *Client) navigateTab(ctx context.Context, t *tab, url string) error {
	return c.runTab(ctx, t, func(page *rod.Page) error {
		if err := page.Navigate(url); err != nil {
			return fmt.Errorf("navigate: %w", err)
		}

		c.waitLoad(page)

		return nil
	})
}

func (c *Client) readMeta(t *tab) pageMeta {
	t.mu.Lock()
	defer t.mu.Unlock()

	return pageMeta{URL: t.url, Title: t.title}
}

func (c *Client) applyMeta(t *tab, meta pageMeta) {
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

	return c.openTab(ctx, sess, url)
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

	_ = s.browser.Close()
}

func (t *tab) close() {
	if t.page == nil {
		return
	}

	_ = t.page.Close()
	t.page = nil
}
