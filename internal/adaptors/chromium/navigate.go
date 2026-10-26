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

	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	return c.navigateTab(ctx, t, url)
}

func (c *Client) Back(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	return c.runTab(ctx, t, func(page *rod.Page) error {
		if err := page.NavigateBack(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}

func (c *Client) Forward(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	return c.runTab(ctx, t, func(page *rod.Page) error {
		if err := page.NavigateForward(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}

func (c *Client) Refresh(ctx context.Context, tabID, sessionKey string) error {
	_, t, err := c.tabFor(sessionKey, tabID)
	if err != nil {
		return err
	}

	return c.runTab(ctx, t, func(page *rod.Page) error {
		if err := page.Reload(); err != nil {
			return err
		}

		c.waitLoad(page)

		return nil
	})
}
