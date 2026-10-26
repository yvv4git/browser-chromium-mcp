package core

import (
	"context"

	"github.com/yvv4git/browser-chromium-mcp/internal/ports"
	"go.uber.org/zap"
)

type Options struct {
	Transport interface{}
	HTTPAddr  string
}

type Server struct {
	browser ports.BrowserClient
	fs      ports.Filesystem
	log     *zap.Logger
	opts    Options
}

func NewServer(browser ports.BrowserClient, fs ports.Filesystem, log *zap.Logger, opts Options) (*Server, error) {
	return &Server{
		browser: browser,
		fs:      fs,
		log:     log,
		opts:    opts,
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	s.log.Info("Starting MCP server")
	<-ctx.Done()
	return nil
}
