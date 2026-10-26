package ports

import (
	"context"

	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
)

type BrowserClient interface {
	Health(ctx context.Context) (*domain.Health, error)
	OpenTab(ctx context.Context, sessionKey, url string) (*domain.Tab, error)
	Navigate(ctx context.Context, tabID, sessionKey, url string) error
	Snapshot(ctx context.Context, tabID, sessionKey string) (*domain.Snapshot, error)
	CloseTab(ctx context.Context, tabID, sessionKey string) error
	CloseSession(ctx context.Context, sessionKey string) error
	Click(ctx context.Context, tabID, sessionKey, ref, selector string) error
	Type(ctx context.Context, tabID, sessionKey, ref, selector, text string) error
	Press(ctx context.Context, tabID, sessionKey, key string) error
	Scroll(ctx context.Context, tabID, sessionKey, direction string, amount int) error
	Back(ctx context.Context, tabID, sessionKey string) error
	Forward(ctx context.Context, tabID, sessionKey string) error
	Refresh(ctx context.Context, tabID, sessionKey string) error
	Links(ctx context.Context, tabID, sessionKey string, limit, offset int) (*domain.Links, error)
	Screenshot(ctx context.Context, tabID, sessionKey string, fullPage bool) ([]byte, error)
	Evaluate(ctx context.Context, tabID, sessionKey, expression string) (*domain.Evaluate, error)
	NetworkRequests(ctx context.Context, tabID, sessionKey string) ([]domain.Resource, error)
	Stats(ctx context.Context, tabID, sessionKey string) (*domain.Stats, error)
	ListTabs(ctx context.Context, sessionKey string) (*domain.ListTabs, error)
}

type NoopBrowserClient struct{}

func (NoopBrowserClient) Health(_ context.Context) (*domain.Health, error) {
	return &domain.Health{}, nil
}

func (NoopBrowserClient) OpenTab(_ context.Context, _, _ string) (*domain.Tab, error) {
	return &domain.Tab{}, nil
}

func (NoopBrowserClient) Navigate(_ context.Context, _, _, _ string) error { return nil }

func (NoopBrowserClient) Snapshot(_ context.Context, _, _ string) (*domain.Snapshot, error) {
	return &domain.Snapshot{}, nil
}

func (NoopBrowserClient) CloseTab(_ context.Context, _, _ string) error   { return nil }
func (NoopBrowserClient) CloseSession(_ context.Context, _ string) error  { return nil }
func (NoopBrowserClient) Click(_ context.Context, _, _, _, _ string) error { return nil }

func (NoopBrowserClient) Type(_ context.Context, _, _, _, _, _ string) error { return nil }

func (NoopBrowserClient) Press(_ context.Context, _, _, _ string) error { return nil }

func (NoopBrowserClient) Scroll(_ context.Context, _, _, _ string, _ int) error { return nil }

func (NoopBrowserClient) Back(_ context.Context, _, _ string) error    { return nil }
func (NoopBrowserClient) Forward(_ context.Context, _, _ string) error { return nil }
func (NoopBrowserClient) Refresh(_ context.Context, _, _ string) error { return nil }

func (NoopBrowserClient) Links(_ context.Context, _, _ string, _, _ int) (*domain.Links, error) {
	return &domain.Links{}, nil
}

func (NoopBrowserClient) Screenshot(_ context.Context, _, _ string, _ bool) ([]byte, error) {
	return nil, nil
}

func (NoopBrowserClient) Evaluate(_ context.Context, _, _, _ string) (*domain.Evaluate, error) {
	return &domain.Evaluate{}, nil
}

func (NoopBrowserClient) NetworkRequests(_ context.Context, _, _ string) ([]domain.Resource, error) {
	return nil, nil
}

func (NoopBrowserClient) Stats(_ context.Context, _, _ string) (*domain.Stats, error) {
	return &domain.Stats{}, nil
}

func (NoopBrowserClient) ListTabs(_ context.Context, _ string) (*domain.ListTabs, error) {
	return &domain.ListTabs{}, nil
}

var _ BrowserClient = NoopBrowserClient{}
