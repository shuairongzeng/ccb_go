package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/daemon"
	"github.com/anthropics/claude_code_bridge/internal/daemon/adapter"
	"github.com/anthropics/claude_code_bridge/internal/protocol"
	ccbruntime "github.com/anthropics/claude_code_bridge/internal/runtime"
)

// AskRequest represents a client-side ask request.
type AskRequest struct {
	Provider string
	Message  string
	WorkDir  string
	TimeoutS float64
	Quiet    bool
	Caller   string
}

// AskResult represents a client-side ask result.
type AskResult struct {
	ExitCode int
	Reply    string
	ReqID    string
	Error    string
}

// Ask sends a request to the daemon and returns the result.
func Ask(req AskRequest) (*AskResult, error) {
	state, err := ReadState("")
	if err != nil {
		// Try to auto-start daemon
		if startErr := MaybeStartDaemon(); startErr != nil {
			return nil, fmt.Errorf("daemon not running and auto-start failed: %w", startErr)
		}
		state, err = ReadState("")
		if err != nil {
			return nil, err
		}
	}

	if req.WorkDir == "" {
		req.WorkDir = ResolveWorkDir(req.Provider)
	}
	if req.TimeoutS == 0 {
		req.TimeoutS = 120
	}

	reqID := protocol.MakeReqID()

	host := ccbruntime.NormalizeConnectHost(state.Host)
	addr := fmt.Sprintf("%s:%d", host, state.Port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon: %w", err)
	}
	defer conn.Close()

	totalTimeout := time.Duration(req.TimeoutS+15) * time.Second
	conn.SetDeadline(time.Now().Add(totalTimeout))

	// Send request
	rpcReq := map[string]interface{}{
		"method":    "request",
		"token":     state.Token,
		"provider":  req.Provider,
		"client_id": fmt.Sprintf("cli-%d", os.Getpid()),
		"work_dir":  req.WorkDir,
		"message":   req.Message,
		"req_id":    reqID,
		"timeout_s": req.TimeoutS,
		"quiet":     req.Quiet,
		"caller":    req.Caller,
	}

	data, _ := json.Marshal(rpcReq)
	conn.Write(data)
	conn.Write([]byte("\n"))

	// Read response
	decoder := json.NewDecoder(conn)
	var result adapter.ProviderResult
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return &AskResult{
		ExitCode: result.ExitCode,
		Reply:    result.Reply,
		ReqID:    result.ReqID,
		Error:    result.Error,
	}, nil
}

// Ping pings a specific provider through the daemon.
func Ping(provider string) error {
	state, err := ReadState("")
	if err != nil {
		return fmt.Errorf("daemon not running")
	}

	resp, err := sendRequest(state, map[string]interface{}{
		"method":   "ping",
		"token":    state.Token,
		"provider": provider,
	})
	if err != nil {
		return err
	}

	status, _ := resp["status"].(string)
	if status != "ok" {
		errMsg, _ := resp["error"].(string)
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

// Pend retrieves the latest reply from a provider.
func Pend(provider string) (string, error) {
	state, err := ReadState("")
	if err != nil {
		return "", fmt.Errorf("daemon not running")
	}

	resp, err := sendRequest(state, map[string]interface{}{
		"method":   "pend",
		"token":    state.Token,
		"provider": provider,
	})
	if err != nil {
		return "", err
	}

	reply, _ := resp["reply"].(string)
	return reply, nil
}

// MaybeStartDaemon starts the daemon if it's not already running.
func MaybeStartDaemon() error {
	// Check if already running
	state, err := ReadState("")
	if err == nil {
		if PingDaemon(state) == nil {
			return nil // Already running
		}
	}

	return MaybeStartDaemonDetached()
}

// MaybeStartDaemonDetached starts the daemon as a detached background process.
// On Windows, uses CREATE_NO_WINDOW / DETACHED_PROCESS flags.
func MaybeStartDaemonDetached() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, "daemon", "start")
	cmd.Dir, _ = os.Getwd()
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Platform-specific detach
	setDetachedProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Detach
	cmd.Process.Release()

	// Wait for daemon to be ready
	return WaitForDaemonReady(10 * time.Second)
}

// WaitForDaemonReady waits for the daemon to become ready.
func WaitForDaemonReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for daemon to start")
		case <-ticker.C:
			state, err := ReadState("")
			if err != nil {
				continue
			}
			if PingDaemon(state) == nil {
				return nil
			}
		}
	}
}

// DetectBackgroundMode detects if the current process is running in background mode.
// Returns true if there's no controlling terminal (PTY).
func DetectBackgroundMode() bool {
	if goruntime.GOOS == "windows" {
		// On Windows, check if we have a console window
		// If TERM is not set and there's no TMUX/WEZTERM, likely background
		if os.Getenv("TERM") == "" && os.Getenv("TMUX") == "" && os.Getenv("WEZTERM_PANE") == "" {
			return true
		}
		return false
	}

	// On Unix, check if stdin is a terminal
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// ResolveWorkDir resolves the working directory from session files or CWD.
func ResolveWorkDir(provider string) string {
	cwd, _ := os.Getwd()

	// Check for session file
	spec := protocol.ClientSpecByPrefix(protocol.ProviderNameMap[provider])
	if spec != nil {
		sessionFile := filepath.Join(cwd, ".ccb_config", spec.SessionFilename)
		if _, err := os.Stat(sessionFile); err == nil {
			return cwd
		}
	}

	return cwd
}

// ResolveWorkDirWithRegistry resolves the working directory using the pane registry.
// Falls back to CWD if no registry match is found.
func ResolveWorkDirWithRegistry(provider string) string {
	cwd, _ := os.Getwd()

	// First try session file
	spec := protocol.ClientSpecByPrefix(protocol.ProviderNameMap[provider])
	if spec != nil {
		sessionFile := filepath.Join(cwd, ".ccb_config", spec.SessionFilename)
		if _, err := os.Stat(sessionFile); err == nil {
			return cwd
		}
	}

	// Try registry file
	registryPath := filepath.Join(ccbruntime.RunDir(), "pane-registry.json")
	if _, err := os.Stat(registryPath); err != nil {
		return cwd
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return cwd
	}

	var registry struct {
		Providers map[string]map[string]struct {
			WorkDir string `json:"work_dir"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return cwd
	}

	if provMap, ok := registry.Providers[provider]; ok {
		for _, entry := range provMap {
			if entry.WorkDir != "" {
				return entry.WorkDir
			}
		}
	}

	return cwd
}

// DaemonStatus returns the daemon status info.
func DaemonStatus() (*daemon.DaemonState, map[string]interface{}, error) {
	state, err := ReadState("")
	if err != nil {
		return nil, nil, err
	}

	status, err := StatusDaemon(state)
	if err != nil {
		return state, nil, err
	}

	return state, status, nil
}

// setDetachedProcess sets platform-specific flags for detached process creation.
func setDetachedProcess(cmd *exec.Cmd) {
	// Platform-specific implementation in client_unix.go and client_windows.go
	setDetachedProcessPlatform(cmd)
}
