package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yvv4git/browser-chromium-mcp/internal/adaptors/chromium"
	fsadapter "github.com/yvv4git/browser-chromium-mcp/internal/adaptors/fs"
	mcpconfig "github.com/yvv4git/browser-chromium-mcp/internal/config/mcp"
	"github.com/yvv4git/browser-chromium-mcp/internal/core"
	"github.com/yvv4git/browser-chromium-mcp/internal/infra"
	"github.com/yvv4git/browser-chromium-mcp/internal/ports"
)

var rootCmd = &cobra.Command{
	Use:   "browser-chromium-mcp",
	Short: "MCP server for Chromium",
}

func Execute() error {
	return rootCmd.Execute()
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("config", "config.toml", "path to config file")
	serveCmd.Flags().String("cdp-url", "", "CDP URL")
	serveCmd.Flags().String("exec-path", "", "path to Chromium executable")
	serveCmd.Flags().Bool("headless", true, "headless mode")
	serveCmd.Flags().String("user-data-dir", "", "user data dir")
	serveCmd.Flags().String("session", "", "default session key")
	serveCmd.Flags().String("transport", "stdio", "transport: stdio or http")
	serveCmd.Flags().String("http-addr", ":8080", "HTTP listen address")
	serveCmd.Flags().Duration("action-timeout", 30*time.Second, "action timeout")
	serveCmd.Flags().Duration("startup-timeout", 30*time.Second, "startup timeout")
	serveCmd.Flags().Int("window-width", 1280, "window width")
	serveCmd.Flags().Int("window-height", 800, "window height")
	serveCmd.Flags().String("log-level", "info", "log level")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath, _ := cmd.Flags().GetString("config")
	cfg := &mcpconfig.Config{}

	if _, err := os.Stat(configPath); err == nil {
		if err := mcpconfig.Load(configPath, cfg); err != nil {
			return err
		}
	} else {
		if err := mcpconfig.LoadEnv(cfg); err != nil {
			return err
		}
	}

	if err := applyFlags(cmd, cfg); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	logger, err := infra.NewWithLogLevel(cfg.Log.Level)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	browserClient, err := chromium.New(ctx, chromium.Options{
		CDPURL:         cfg.Serve.CDPURL,
		ExecPath:       cfg.Serve.ExecPath,
		Headless:       cfg.Serve.Headless,
		UserDataDir:    cfg.Serve.UserDataDir,
		WindowWidth:    cfg.Serve.WindowWidth,
		WindowHeight:   cfg.Serve.WindowHeight,
		ActionTimeout:  cfg.Serve.ActionTimeout,
		StartupTimeout: cfg.Serve.StartupTimeout,
	}, logger.Named("chromium"))
	if err != nil {
		return err
	}
	defer func() { _ = browserClient.Close() }()

	fs := ports.Filesystem(fsadapter.New())

	srv, err := core.NewServer(browserClient, fs, logger.Named("mcp"), core.Options{
		Transport: cfg.Serve.Transport,
		HTTPAddr:  cfg.Serve.HTTPAddr,
	})
	if err != nil {
		return err
	}

	if err := srv.Serve(ctx); err != nil {
		return err
	}

	return nil
}

func applyFlags(cmd *cobra.Command, cfg *mcpconfig.Config) error {
	if cmd.Flags().Changed("cdp-url") {
		v, _ := cmd.Flags().GetString("cdp-url")
		cfg.Serve.CDPURL = v
	}
	if cmd.Flags().Changed("exec-path") {
		v, _ := cmd.Flags().GetString("exec-path")
		cfg.Serve.ExecPath = v
	}
	if cmd.Flags().Changed("headless") {
		v, _ := cmd.Flags().GetBool("headless")
		cfg.Serve.Headless = v
	}
	if cmd.Flags().Changed("user-data-dir") {
		v, _ := cmd.Flags().GetString("user-data-dir")
		cfg.Serve.UserDataDir = v
	}
	if cmd.Flags().Changed("session") {
		v, _ := cmd.Flags().GetString("session")
		cfg.Serve.Session = v
	}
	if cmd.Flags().Changed("transport") {
		v, _ := cmd.Flags().GetString("transport")
		t, err := configParseTransport(v)
		if err != nil {
			return err
		}
		cfg.Serve.Transport = t
	}
	if cmd.Flags().Changed("http-addr") {
		v, _ := cmd.Flags().GetString("http-addr")
		cfg.Serve.HTTPAddr = v
	}
	if cmd.Flags().Changed("action-timeout") {
		v, _ := cmd.Flags().GetDuration("action-timeout")
		cfg.Serve.ActionTimeout = v
	}
	if cmd.Flags().Changed("startup-timeout") {
		v, _ := cmd.Flags().GetDuration("startup-timeout")
		cfg.Serve.StartupTimeout = v
	}
	if cmd.Flags().Changed("window-width") {
		v, _ := cmd.Flags().GetInt("window-width")
		cfg.Serve.WindowWidth = v
	}
	if cmd.Flags().Changed("window-height") {
		v, _ := cmd.Flags().GetInt("window-height")
		cfg.Serve.WindowHeight = v
	}
	if cmd.Flags().Changed("log-level") {
		v, _ := cmd.Flags().GetString("log-level")
		cfg.Log.Level = v
	}
	return nil
}


