package chromium

import (
	"context"

	"github.com/chromedp/chromedp"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
)

func (c *Client) Snapshot(ctx context.Context, tabID, sessionKey string) (*domain.Snapshot, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	var url string
	var body string
	if err := c.runTab(ctx, t, chromedp.Location(&url), chromedp.Evaluate(_snapshotJS, &body)); err != nil {
		return nil, err
	}

	t.mu.Lock()
	t.refsCount = countRefs(body)
	t.mu.Unlock()

	return &domain.Snapshot{
		URL:       url,
		Snapshot:  body,
		RefsCount: t.refsCount,
	}, nil
}

func (c *Client) Click(ctx context.Context, tabID, sessionKey, ref, selector string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	sel, err := c.elementSelector(ref, selector)
	if err != nil {
		return err
	}

	if err := c.runTab(ctx, t, chromedp.WaitVisible(sel, chromedp.ByQuery), chromedp.Click(sel, chromedp.ByQuery)); err != nil {
		return err
	}

	return nil
}

func (c *Client) Type(ctx context.Context, tabID, sessionKey, ref, selector, text string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	sel, err := c.elementSelector(ref, selector)
	if err != nil {
		return err
	}

	if err := c.runTab(ctx, t, chromedp.WaitVisible(sel, chromedp.ByQuery), chromedp.SendKeys(sel, text, chromedp.ByQuery)); err != nil {
		return err
	}

	return nil
}

func (c *Client) Press(ctx context.Context, tabID, sessionKey, key string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	k, err := mapKey(key)
	if err != nil {
		return err
	}

	if err := c.runTab(ctx, t, chromedp.KeyEvent(k)); err != nil {
		return err
	}

	return nil
}

func (c *Client) Scroll(ctx context.Context, tabID, sessionKey, direction string, amount int) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	script, err := scrollScript(direction, amount)
	if err != nil {
		return err
	}

	if err := c.runTab(ctx, t, chromedp.Evaluate(script, nil)); err != nil {
		return err
	}

	return nil
}

func (c *Client) Links(ctx context.Context, tabID, sessionKey string, limit, offset int) (*domain.Links, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var all []domain.Link
	if err := c.runTab(ctx, t, chromedp.Evaluate(_linksJS, &all)); err != nil {
		return nil, err
	}

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	links := make([]domain.Link, 0, end-offset)
	for i := offset; i < end; i++ {
		links = append(links, all[i])
	}

	return &domain.Links{
		Links:   links,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: end < total,
	}, nil
}

func (c *Client) Screenshot(ctx context.Context, tabID, sessionKey string, fullPage bool) ([]byte, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	var buf []byte
	if fullPage {
		if err := c.runTab(ctx, t, chromedp.FullScreenshot(&buf, 90)); err != nil {
			return nil, err
		}
		return buf, nil
	}

	if err := c.runTab(ctx, t, chromedp.CaptureScreenshot(&buf)); err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *Client) Evaluate(ctx context.Context, tabID, sessionKey, expression string) (*domain.Evaluate, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	var res any
	if err := c.runTab(ctx, t, chromedp.Evaluate(expression, &res)); err != nil {
		return &domain.Evaluate{OK: false, Result: nil}, err
	}

	return &domain.Evaluate{OK: true, Result: res}, nil
}

func (c *Client) NetworkRequests(ctx context.Context, tabID, sessionKey string) ([]domain.Resource, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	if err := c.runTab(ctx, t, chromedp.ActionFunc(func(ctx context.Context) error {
		return nil
	})); err != nil {
		return nil, err
	}

	return []domain.Resource{}, nil
}

func (c *Client) Stats(ctx context.Context, tabID, sessionKey string) (*domain.Stats, error) {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return &domain.Stats{
		TabID:       t.id,
		URL:         t.url,
		Title:       t.title,
		VisitedURLs: append([]string(nil), t.visited...),
		ToolCalls:   t.toolCalls,
		RefsCount:   t.refsCount,
	}, nil
}
