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
	key       string
	gen       uint64
	contextID proto.BrowserBrowserContextID
	mu        sync.Mutex
	tabs      map[string]*tab
	lastTabID string
}

type tab struct {
	id        string
	mu        sync.Mutex
	url       string
	title     string
	visited   []string
	toolCalls int
	refsCount int
}

func (c *Client) newSession(ctx context.Context, cn *conn, key string) (*session, error) {
	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
	defer cancel()

	res, err := proto.TargetCreateBrowserContext{}.Call(cn.browser.Context(runCtx))
	if err != nil {
		return nil, fmt.Errorf("create browser context: %w", err)
	}

	c.log.Debug("Session created", zap.String("session", key))

	return &session{
		key:       key,
		gen:       cn.gen,
		contextID: res.BrowserContextID,
		tabs:      make(map[string]*tab),
	}, nil
}

func (c *Client) pageMeta(page *rod.Page) pageMeta {
	info, err := page.Info()
	if err != nil {
		return pageMeta{}
	}

	return pageMeta{URL: info.URL, Title: info.Title}
}

func (c *Client) openTab(ctx context.Context, cn *conn, sess *session, url string) (*domain.Tab, error) {
	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
	defer cancel()

	sessBrowser := *cn.browser
	sessBrowser.BrowserContextID = sess.contextID

	page, err := sessBrowser.Context(runCtx).Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("open tab: %w", err)
	}

	t := &tab{id: string(page.TargetID)}

	if err := c.navigatePage(page, url); err != nil {
		_, _ = proto.TargetCloseTarget{TargetID: page.TargetID}.Call(cn.browser)

		return nil, fmt.Errorf("open tab: %w", err)
	}

	meta := c.pageMeta(page)
	t.url = meta.URL
	t.title = meta.Title
	t.toolCalls = 1

	if t.url != "" {
		t.visited = append(t.visited, t.url)
	}

	if err := sess.addTab(t); err != nil {
		_, _ = proto.TargetCloseTarget{TargetID: page.TargetID}.Call(cn.browser)

		return nil, err
	}

	c.log.Debug("Tab opened",
		zap.String("session", sess.key),
		zap.String("tab", t.id),
		zap.String("url", url),
	)

	return &domain.Tab{TabID: t.id, TargetID: t.id, URL: t.url, Title: t.title}, nil
}

func (c *Client) runTab(ctx context.Context, sessionKey, tabID string, action func(*tab, *rod.Page) error) error {
	cn, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer cn.close()

	_, t, err := c.tabFor(cn, sessionKey, tabID)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.toolCalls++

	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
	defer cancel()

	page, err := cn.browser.Context(runCtx).PageFromTarget(proto.TargetTargetID(t.id))
	if err != nil {
		return fmt.Errorf("attach tab: %w", err)
	}

	if err := action(t, page); err != nil {
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

func (c *Client) navigatePage(page *rod.Page, url string) error {
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}

	c.waitLoad(page)

	return nil
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
	cn, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer cn.close()

	sess, err := c.ensureSession(ctx, cn, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}

	return c.openTab(ctx, cn, sess, url)
}

func (c *Client) CloseTab(ctx context.Context, sessionKey, tabID string) error {
	cn, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer cn.close()

	sess, t, err := c.tabFor(cn, sessionKey, tabID)
	if err != nil {
		return err
	}

	_, _ = sess.removeTab(tabID)

	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
	defer cancel()

	if _, err := (proto.TargetCloseTarget{TargetID: proto.TargetTargetID(t.id)}).Call(cn.browser.Context(runCtx)); err != nil {
		return fmt.Errorf("close tab: %w", err)
	}

	c.log.Debug("Tab closed", zap.String("session", sessionKey), zap.String("tab", tabID))

	return nil
}

func (c *Client) CloseSession(ctx context.Context, sessionKey string) error {
	cn, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer cn.close()

	sess := c.removeSession(sessionKey)
	if sess == nil || sess.gen != cn.gen {
		return domain.ErrSessionNotFound
	}

	runCtx, cancel := context.WithTimeout(ctx, c.options.actionTimeout())
	defer cancel()

	sessBrowser := *cn.browser
	sessBrowser.BrowserContextID = sess.contextID

	if err := sessBrowser.Context(runCtx).Close(); err != nil {
		return fmt.Errorf("close session: %w", err)
	}

	c.log.Debug("Session closed", zap.String("session", sessionKey))

	return nil
}

func (c *Client) ListTabs(ctx context.Context, sessionKey string) (*domain.ListTabs, error) {
	cn, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	cn.close()

	sess, err := c.getSession(sessionKey)
	if err != nil {
		return nil, err
	}

	if sess.gen != cn.gen {
		return nil, domain.ErrSessionNotFound
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
	s.lastTabID = t.id

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

	if s.lastTabID == tabID {
		s.lastTabID = ""

		for id := range s.tabs {
			s.lastTabID = id

			break
		}
	}

	return t, nil
}

func (s *session) tab(tabID string) (*tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tabID == "" {
		tabID = s.lastTabID
	}

	t, ok := s.tabs[tabID]
	if !ok {
		return nil, domain.ErrTabNotFound
	}

	s.lastTabID = tabID

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
