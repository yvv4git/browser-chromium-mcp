package chromium

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
)

func (c *Client) Snapshot(ctx context.Context, tabID, sessionKey string) (*domain.Snapshot, error) {
	var (
		body     string
		refs     int
		snapshot string
	)

	if err := c.runTab(ctx, sessionKey, tabID, func(t *tab, page *rod.Page) error {
		res, err := page.Eval(_snapshotJS)
		if err != nil {
			return err
		}

		body = res.Value.Str()
		refs = countRefs(body)
		t.refsCount = refs
		snapshot = c.pageMeta(page).URL

		return nil
	}); err != nil {
		return nil, err
	}

	return &domain.Snapshot{
		URL:       snapshot,
		Snapshot:  body,
		RefsCount: refs,
	}, nil
}

func (c *Client) element(page *rod.Page, selector string) (*rod.Element, error) {
	el, err := page.Element(selector)
	if err != nil {
		var notFound *rod.ElementNotFoundError
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("%w: %s", domain.ErrElementNotFound, selector)
		}

		return nil, err
	}

	return el, nil
}

func (c *Client) Click(ctx context.Context, tabID, sessionKey, ref, selector string) error {
	sel, err := c.elementSelector(ref, selector)
	if err != nil {
		return err
	}

	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		el, err := c.element(page, sel)
		if err != nil {
			return err
		}

		return el.Click(proto.InputMouseButtonLeft, 1)
	})
}

func (c *Client) Type(ctx context.Context, tabID, sessionKey, ref, selector, text string) error {
	sel, err := c.elementSelector(ref, selector)
	if err != nil {
		return err
	}

	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		el, err := c.element(page, sel)
		if err != nil {
			return err
		}

		return el.Input(text)
	})
}

func (c *Client) Press(ctx context.Context, tabID, sessionKey, key string) error {
	k, err := mapKey(key)
	if err != nil {
		return err
	}

	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		return page.Keyboard.Type(k)
	})
}

func (c *Client) Scroll(ctx context.Context, tabID, sessionKey, direction string, amount int) error {
	script, err := scrollScript(direction, amount)
	if err != nil {
		return err
	}

	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		_, err := page.Eval(script)

		return err
	})
}

func (c *Client) Links(ctx context.Context, tabID, sessionKey string, limit, offset int) (*domain.Links, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var all []domain.Link

	if err := c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		res, err := page.Eval(_linksJS)
		if err != nil {
			return err
		}

		return res.Value.Unmarshal(&all)
	}); err != nil {
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
	var data []byte

	if err := c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		shot, err := page.Screenshot(fullPage, nil)
		if err != nil {
			return err
		}

		data = shot

		return nil
	}); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) Evaluate(ctx context.Context, tabID, sessionKey, expression string) (*domain.Evaluate, error) {
	var result any

	if err := c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		res, err := proto.RuntimeEvaluate{
			Expression:    expression,
			ReturnByValue: true,
			AwaitPromise:  true,
			UserGesture:   true,
		}.Call(page)
		if err != nil {
			return err
		}

		result = res.Result.Value.Val()

		return nil
	}); err != nil {
		return &domain.Evaluate{OK: false, Result: nil}, err
	}

	return &domain.Evaluate{OK: true, Result: result}, nil
}

func (c *Client) NetworkRequests(ctx context.Context, tabID, sessionKey string) ([]domain.Resource, error) {
	if err := c.runTab(ctx, sessionKey, tabID, func(_ *tab, _ *rod.Page) error {
		return nil
	}); err != nil {
		return nil, err
	}

	return []domain.Resource{}, nil
}

func (c *Client) Stats(ctx context.Context, tabID, sessionKey string) (*domain.Stats, error) {
	var stats *domain.Stats

	if err := c.runTab(ctx, sessionKey, tabID, func(t *tab, _ *rod.Page) error {
		stats = &domain.Stats{
			TabID:       t.id,
			URL:         t.url,
			Title:       t.title,
			VisitedURLs: append([]string(nil), t.visited...),
			ToolCalls:   t.toolCalls,
			RefsCount:   t.refsCount,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return stats, nil
}
