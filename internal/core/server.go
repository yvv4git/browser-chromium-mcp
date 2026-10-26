package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	mcp     *server.MCPServer
}

func NewServer(browser ports.BrowserClient, fs ports.Filesystem, log *zap.Logger, opts Options) (*Server, error) {
	mcpServer := server.NewMCPServer(
		"browser-chromium-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s := &Server{
		browser: browser,
		fs:      fs,
		log:     log,
		opts:    opts,
		mcp:     mcpServer,
	}

	s.registerTools()

	return s, nil
}

func (s *Server) registerTools() {
	// Navigation & tabs
	s.addTool("health", "Get browser health status", map[string]*jsonSchema{}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := s.browser.Health(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("open_tab", "Open a new tab in a session", map[string]*jsonSchema{
		"sessionKey": {Type: "string"},
		"url":        {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.OpenTab(ctx, getString(args, "sessionKey"), getString(args, "url"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("navigate", "Navigate to URL", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"url":        {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Navigate(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "url"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("back", "Navigate back", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Back(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("forward", "Navigate forward", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Forward(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("refresh", "Refresh page", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Refresh(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("list_tabs", "List tabs in session", map[string]*jsonSchema{
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.ListTabs(ctx, getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("close_tab", "Close tab", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.CloseTab(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("close_session", "Close session", map[string]*jsonSchema{
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.CloseSession(ctx, getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	// Actions & data
	s.addTool("snapshot", "Get ARIA snapshot with refs", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.Snapshot(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("click", "Click element", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"ref":        {Type: "string"},
		"selector":   {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Click(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "ref"), getString(args, "selector"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("type", "Type text into element", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"ref":        {Type: "string"},
		"selector":   {Type: "string"},
		"text":       {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Type(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "ref"), getString(args, "selector"), getString(args, "text"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("press", "Press key", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"key":        {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Press(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "key"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("scroll", "Scroll page", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"direction":  {Type: "string"},
		"amount":     {Type: "number"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		err := s.browser.Scroll(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "direction"), getInt(args, "amount"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})

	s.addTool("links", "Get links on page", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"limit":      {Type: "number"},
		"offset":     {Type: "number"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.Links(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getInt(args, "limit"), getInt(args, "offset"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("screenshot", "Take screenshot", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"fullPage":   {Type: "boolean"},
		"path":       {Type: "string"},
		"quality":    {Type: "number"},
		"maxWidth":   {Type: "number"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		data, err := s.browser.Screenshot(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getBool(args, "fullPage"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if p := getString(args, "path"); p != "" {
			if err := s.fs.Save(p, data); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(mustJSON(map[string]any{"ok": true, "path": p})), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.ImageContent{
					Type:     "image",
					Data:     base64.StdEncoding.EncodeToString(data),
					MIMEType: "image/png",
				},
			},
		}, nil
	})

	s.addTool("evaluate", "Evaluate JavaScript", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
		"expression": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.Evaluate(ctx, getString(args, "tabId"), getString(args, "sessionKey"), getString(args, "expression"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("network_requests", "Get network requests", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.NetworkRequests(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})

	s.addTool("stats", "Get tab stats", map[string]*jsonSchema{
		"tabId":      {Type: "string"},
		"sessionKey": {Type: "string"},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		res, err := s.browser.Stats(ctx, getString(args, "tabId"), getString(args, "sessionKey"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(mustJSON(res)), nil
	})
}

func (s *Server) addTool(name, description string, schema map[string]*jsonSchema, handler server.ToolHandlerFunc) {
	props := map[string]any{}
	for k, v := range schema {
		props[k] = map[string]string{"type": v.Type}
	}

	_ = mcp.NewTool(name,
		mcp.WithDescription(description),
		mcp.WithObject("params"),
	)

	// build simple tool
	t := mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: props,
		},
	}
	s.mcp.AddTool(t, handler)
}

type jsonSchema struct {
	Type string
}

func (s *Server) Serve(ctx context.Context) error {
	transport := fmt.Sprintf("%v", s.opts.Transport)
	if transport == "http" || transport == "http" {
		addr := s.opts.HTTPAddr
		if addr == "" {
			addr = ":8080"
		}

		httpServer := server.NewStreamableHTTPServer(s.mcp)
		srv := &http.Server{
			Addr:              addr,
			Handler:           httpServer,
			ReadHeaderTimeout: 5 * time.Second,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()

		s.log.Info("Starting MCP server", zap.String("transport", "http"), zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}

	s.log.Info("Starting MCP server", zap.String("transport", "stdio"))
	return server.ServeStdio(s.mcp)
}
