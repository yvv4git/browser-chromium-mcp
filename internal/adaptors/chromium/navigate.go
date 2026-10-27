package chromium

import (
	"context"

	"github.com/go-rod/rod"
)

const _defaultSessionKey = "default"

func (c *Client) Navigate(ctx context.Context, tabID, sessionKey, url string) error {
	if tabID == "" {
		if sessionKey == "" {
			sessionKey = _defaultSessionKey
		}

		_, err := c.OpenTab(ctx, sessionKey, url)

		return err
	}

	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		return c.navigatePage(page, url)
	})
}

func (c *Client) Back(ctx context.Context, tabID, sessionKey string) error {
	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		if err := page.NavigateBack(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}

func (c *Client) Forward(ctx context.Context, tabID, sessionKey string) error {
	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		if err := page.NavigateForward(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}

func (c *Client) Refresh(ctx context.Context, tabID, sessionKey string) error {
	return c.runTab(ctx, sessionKey, tabID, func(_ *tab, page *rod.Page) error {
		if err := page.Reload(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}
