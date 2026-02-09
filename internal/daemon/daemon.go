package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/config"
	"github.com/anthropics/claude_code_bridge/internal/daemon/adapter"
	"github.com/anthropics/claude_code_bridge/internal/runtime"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// UnifiedDaemon manages the lifecycle of the unified ask daemon.
type UnifiedDaemon struct {
	server   *Server
	registry *Registry
	backend  terminal.Backend
}

// DaemonConfig holds configuration for the unified daemon.
type DaemonConfig struct {
	Host        string
	Port        int
	Providers   []string
	IdleTimeout time.Duration
	ParentPID   int
	StateFile   string
	LogFile     string
}

// NewUnifiedDaemon creates a new unified daemon.
func NewUnifiedDaemon(cfg DaemonConfig) (*UnifiedDaemon, error) {
	// Detect terminal backend
	backend, err := terminal.DetectBackend()
	if err != nil {
		// On Windows, PowerShell backend might not be "available" in the traditional sense
		// but we still want to run the daemon
		backend = nil
	}

	// Create registry and register adapters
	registry := NewRegistry()

	for _, provider := range cfg.Providers {
		var a adapter.Adapter
		switch provider {
		case "codex":
			a = adapter.NewCodexAdapter(backend)
		case "gemini":
			a = adapter.NewGeminiAdapter(backend)
		case "opencode":
			a = adapter.NewOpenCodeAdapter(backend)
		case "claude":
			a = adapter.NewClaudeAdapter(backend)
		case "droid":
			a = adapter.NewDroidAdapter(backend)
		default:
			continue
		}
		registry.Register(provider, a)
	}

	// Determine state and log files
	if cfg.StateFile == "" {
		cfg.StateFile = runtime.StateFilePath("askd")
	}
	if cfg.LogFile == "" {
		cfg.LogFile = runtime.LogPath("askd")
	}

	server := NewServer(ServerConfig{
		Host:        cfg.Host,
		Port:        cfg.Port,
		StateFile:   cfg.StateFile,
		LogFile:     cfg.LogFile,
		IdleTimeout: cfg.IdleTimeout,
		ParentPID:   cfg.ParentPID,
	}, registry)

	return &UnifiedDaemon{
		server:   server,
		registry: registry,
		backend:  backend,
	}, nil
}

// Run starts the daemon and blocks until shutdown.
func (d *UnifiedDaemon) Run() error {
	host := "127.0.0.1"
	port := 0 // auto-assign

	if err := d.server.Start(host, port); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "received signal %v, shutting down\n", sig)
		d.server.Shutdown()
	case <-d.server.shutdown:
		// Already shutting down
	}

	d.server.Wait()
	return nil
}

// RunDefault creates and runs a daemon with default configuration.
func RunDefault() error {
	cwd, _ := os.Getwd()
	cfg := LoadStartConfig(cwd)
	providers := cfg.GetProviders()

	idleTimeout := time.Duration(config.EnvInt("CCB_ASKD_IDLE_TIMEOUT_S", 1800)) * time.Second

	daemon, err := NewUnifiedDaemon(DaemonConfig{
		Providers:   providers,
		IdleTimeout: idleTimeout,
		ParentPID:   os.Getppid(),
	})
	if err != nil {
		return err
	}

	return daemon.Run()
}

// LoadStartConfig loads the start configuration for the daemon.
func LoadStartConfig(workDir string) *config.StartConfig {
	return config.LoadStartConfig(workDir)
}
