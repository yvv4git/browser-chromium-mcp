package chromium

import (
	"context"

	"github.com/chromedp/chromedp"
)

func (c *Client) Navigate(ctx context.Context, tabID, sessionKey, url string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	var meta pageMeta
	if err := c.runTab(ctx, t, chromedp.Navigate(url), chromedp.Evaluate(_pageMetaJS, &meta)); err != nil {
		return err
	}

	c.updateMeta(t, meta)

	return nil
}

func (c *Client) Back(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	var meta pageMeta
	if err := c.runTab(ctx, t, chromedp.NavigateBack(), chromedp.Evaluate(_pageMetaJS, &meta)); err != nil {
		return err
	}

	c.updateMeta(t, meta)

	return nil
}

func (c *Client) Forward(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	var meta pageMeta
	if err := c.runTab(ctx, t, chromedp.NavigateForward(), chromedp.Evaluate(_pageMetaJS, &meta)); err != nil {
		return err
	}

	c.updateMeta(t, meta)

	return nil
}

func (c *Client) Refresh(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	var meta pageMeta
	if err := c.runTab(ctx, t, chromedp.Reload(), chromedp.Evaluate(_pageMetaJS, &meta)); err != nil {
		return err
	}

	c.updateMeta(t, meta)

	return nil
}
