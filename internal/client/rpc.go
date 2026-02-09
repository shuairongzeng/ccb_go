package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/daemon"
	"github.com/anthropics/claude_code_bridge/internal/runtime"
)

// ReadState reads the daemon state from the state file.
func ReadState(stateFile string) (*daemon.DaemonState, error) {
	if stateFile == "" {
		stateFile = runtime.StateFilePath("askd")
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("daemon not running (no state file)")
	}
	var state daemon.DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("invalid state file: %w", err)
	}
	return &state, nil
}

// PingDaemon sends a ping to the daemon.
func PingDaemon(state *daemon.DaemonState) error {
	resp, err := sendRequest(state, map[string]interface{}{
		"method": "ping",
		"token":  state.Token,
	})
	if err != nil {
		return err
	}
	status, _ := resp["status"].(string)
	if status != "ok" {
		errMsg, _ := resp["error"].(string)
		return fmt.Errorf("ping failed: %s", errMsg)
	}
	return nil
}

// ShutdownDaemon sends a shutdown request to the daemon.
func ShutdownDaemon(state *daemon.DaemonState) error {
	_, err := sendRequest(state, map[string]interface{}{
		"method": "shutdown",
		"token":  state.Token,
	})
	return err
}

// StatusDaemon gets the daemon status.
func StatusDaemon(state *daemon.DaemonState) (map[string]interface{}, error) {
	return sendRequest(state, map[string]interface{}{
		"method": "status",
		"token":  state.Token,
	})
}

// sendRequest sends a JSON request to the daemon and returns the response.
func sendRequest(state *daemon.DaemonState, req map[string]interface{}) (map[string]interface{}, error) {
	host := runtime.NormalizeConnectHost(state.Host)
	addr := fmt.Sprintf("%s:%d", host, state.Port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon at %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Send request
	data, _ := json.Marshal(req)
	conn.Write(data)
	conn.Write([]byte("\n"))

	// Read response
	decoder := json.NewDecoder(conn)
	var resp map[string]interface{}
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	return resp, nil
}
