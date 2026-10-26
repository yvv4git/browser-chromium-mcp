package chromium

import (
	"context"
	"runtime"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

const (
	_flagHeadless = "headless"
	_osDarwin     = "darwin"
)

type Options struct {
	CDPURL         string
	ExecPath       string
	Headless       bool
	UserDataDir    string
	WindowWidth    int
	WindowHeight   int
	ActionTimeout  time.Duration
	StartupTimeout time.Duration
}

func (o Options) allocatorContext(parent context.Context) (context.Context, context.CancelFunc) {
	if o.CDPURL != "" {
		allocCtx, _ := chromedp.NewRemoteAllocator(parent, o.CDPURL)
		return allocCtx, func() {}
	}

	return chromedp.NewExecAllocator(parent, o.execAllocatorOptions()...)
}

func (o Options) execAllocatorOptions() []chromedp.ExecAllocatorOption {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.WindowSize(o.WindowWidth, o.WindowHeight))

	if o.ExecPath != "" {
		options = append(options, chromedp.ExecPath(o.ExecPath))
	}

	if o.UserDataDir != "" {
		options = append(options, chromedp.UserDataDir(o.UserDataDir))
	}

	if !o.Headless {
		options = append(options, chromedp.Flag(_flagHeadless, false))
	}

	return options
}

func (Options) selectAllModifier() input.Modifier {
	if runtime.GOOS == _osDarwin {
		return input.ModifierMeta
	}

	return input.ModifierCtrl
}
