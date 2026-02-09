package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/daemon/adapter"
	"github.com/anthropics/claude_code_bridge/internal/runtime"
)

// Server implements a TCP JSON-RPC server for the unified ask daemon.
type Server struct {
	listener    net.Listener
	token       string
	registry    *Registry
	workerPool  *WorkerPool
	mu          sync.Mutex
	lastActive  time.Time
	idleTimeout time.Duration
	stateFile   string
	logFile     string
	parentPID   int
	shutdown    chan struct{}
	done        chan struct{}
}

// ServerConfig holds configuration for the daemon server.
type ServerConfig struct {
	Host        string
	Port        int
	Token       string
	StateFile   string
	LogFile     string
	IdleTimeout time.Duration
	ParentPID   int
}

// DaemonState represents the persisted daemon state.
type DaemonState struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

// NewServer creates a new daemon server.
func NewServer(cfg ServerConfig, registry *Registry) *Server {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.Token == "" {
		cfg.Token = runtime.RandomToken()
	}

	return &Server{
		token:       cfg.Token,
		registry:    registry,
		workerPool:  NewWorkerPool(50),
		lastActive:  time.Now(),
		idleTimeout: cfg.IdleTimeout,
		stateFile:   cfg.StateFile,
		logFile:     cfg.LogFile,
		parentPID:   cfg.ParentPID,
		shutdown:    make(chan struct{}),
		done:        make(chan struct{}),
	}
}

// Start starts the daemon server.
func (s *Server) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Try with port 0 for auto-assignment
		listener, err = net.Listen("tcp", fmt.Sprintf("%s:0", host))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}
	s.listener = listener

	// Get actual port
	actualAddr := listener.Addr().(*net.TCPAddr)
	actualPort := actualAddr.Port

	// Write state file
	s.writeState(host, actualPort)

	s.log("daemon started on %s:%d (pid=%d)", host, actualPort, os.Getpid())

	// Start idle monitor
	go s.idleMonitor()

	// Start parent process monitor
	if s.parentPID > 0 {
		go s.parentMonitor()
	}

	// Accept connections
	go s.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return
			default:
				s.log("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// handleConn handles a single client connection.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	decoder := json.NewDecoder(conn)
	var req map[string]interface{}
	if err := decoder.Decode(&req); err != nil {
		s.sendError(conn, "invalid request")
		return
	}

	// Verify token
	token, _ := req["token"].(string)
	if token != s.token {
		s.sendError(conn, "invalid token")
		return
	}

	s.touchActivity()

	method, _ := req["method"].(string)
	switch method {
	case "ping", ".ping":
		s.handlePing(conn, req)
	case "shutdown", ".shutdown":
		s.handleShutdown(conn)
	case "status", ".status":
		s.handleStatus(conn)
	case "request", ".request", "ask":
		s.handleRequest(conn, req)
	case "pend", ".pend":
		s.handlePend(conn, req)
	default:
		s.sendError(conn, fmt.Sprintf("unknown method: %s", method))
	}
}

// handlePing handles a ping request.
func (s *Server) handlePing(conn net.Conn, req map[string]interface{}) {
	provider, _ := req["provider"].(string)
	if provider != "" {
		a, ok := s.registry.Get(provider)
		if !ok {
			s.sendJSON(conn, map[string]interface{}{"status": "error", "error": "unknown provider: " + provider})
			return
		}
		sessionID, _ := req["session_id"].(string)
		if err := a.Ping(context.Background(), sessionID); err != nil {
			s.sendJSON(conn, map[string]interface{}{"status": "error", "error": err.Error()})
			return
		}
	}
	s.sendJSON(conn, map[string]interface{}{"status": "ok", "providers": s.registry.Names()})
}

// handleShutdown handles a shutdown request.
func (s *Server) handleShutdown(conn net.Conn) {
	s.sendJSON(conn, map[string]interface{}{"status": "ok", "message": "shutting down"})
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.Shutdown()
	}()
}

// handleStatus handles a status request.
func (s *Server) handleStatus(conn net.Conn) {
	s.sendJSON(conn, map[string]interface{}{
		"status":         "ok",
		"pid":            os.Getpid(),
		"providers":      s.registry.Names(),
		"workers":        s.workerPool.ActiveWorkers(),
		"active_requests": s.activeRequestCount(),
	})
}

// handlePend handles a pend request (retrieve latest reply from a provider).
func (s *Server) handlePend(conn net.Conn, req map[string]interface{}) {
	provider, _ := req["provider"].(string)
	if provider == "" {
		s.sendError(conn, "missing provider")
		return
	}

	a, ok := s.registry.Get(provider)
	if !ok {
		s.sendError(conn, "unknown provider: "+provider)
		return
	}

	sessionID, _ := req["session_id"].(string)
	reply, err := a.Pend(context.Background(), sessionID)
	if err != nil {
		s.sendJSON(conn, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
			"reply":  "",
		})
		return
	}

	s.sendJSON(conn, map[string]interface{}{
		"status": "ok",
		"reply":  reply,
	})
}

// activeRequestCount returns the number of active workers processing requests.
func (s *Server) activeRequestCount() int {
	return s.workerPool.ActiveWorkers()
}

// handleRequest handles an ask request.
func (s *Server) handleRequest(conn net.Conn, req map[string]interface{}) {
	provider, _ := req["provider"].(string)
	if provider == "" {
		s.sendError(conn, "missing provider")
		return
	}

	a, ok := s.registry.Get(provider)
	if !ok {
		s.sendError(conn, "unknown provider: "+provider)
		return
	}

	// Build provider request
	provReq := &adapter.ProviderRequest{
		ClientID: getStr(req, "client_id"),
		WorkDir:  getStr(req, "work_dir"),
		Message:  getStr(req, "message"),
		ReqID:    getStr(req, "req_id"),
		TimeoutS: getFloat(req, "timeout_s"),
		Quiet:    getBool(req, "quiet"),
		Caller:   getStr(req, "caller"),
	}

	// Execute via worker pool
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(provReq.TimeoutS+10)*time.Second)
	task := &adapter.QueuedTask{
		Request:  provReq,
		ResultCh: make(chan *adapter.ProviderResult, 1),
		Ctx:      ctx,
		Cancel:   cancel,
	}

	sessionKey := fmt.Sprintf("%s:%s", provider, provReq.WorkDir)
	s.workerPool.Submit(sessionKey, task, func(taskCtx context.Context, t *adapter.QueuedTask) {
		result, err := a.Send(t.Ctx, t.Request)
		if err != nil {
			t.ResultCh <- &adapter.ProviderResult{ExitCode: 1, Error: err.Error(), ReqID: t.Request.ReqID}
		} else {
			t.ResultCh <- result
		}
	})

	// Wait for result
	select {
	case result := <-task.ResultCh:
		cancel()
		s.sendJSON(conn, result)
	case <-ctx.Done():
		cancel()
		s.sendJSON(conn, &adapter.ProviderResult{ExitCode: 2, Error: "timeout", ReqID: provReq.ReqID})
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	s.log("shutting down...")
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}
	s.workerPool.Shutdown()
	s.removeState()
}

// Wait waits for the server to finish.
func (s *Server) Wait() {
	<-s.done
}

// Addr returns the listener address.
func (s *Server) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// Token returns the server token.
func (s *Server) Token() string {
	return s.token
}

// touchActivity updates the last activity timestamp.
func (s *Server) touchActivity() {
	s.mu.Lock()
	s.lastActive = time.Now()
	s.mu.Unlock()
}

// idleMonitor shuts down the server after idle timeout.
func (s *Server) idleMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			s.mu.Lock()
			idle := time.Since(s.lastActive)
			s.mu.Unlock()
			if idle > s.idleTimeout {
				s.log("idle timeout (%v), shutting down", s.idleTimeout)
				s.Shutdown()
				return
			}
		}
	}
}

// parentMonitor shuts down if the parent process dies.
func (s *Server) parentMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			proc, err := os.FindProcess(s.parentPID)
			if err != nil {
				s.log("parent process %d gone, shutting down", s.parentPID)
				s.Shutdown()
				return
			}
			// On Unix, FindProcess always succeeds; check with signal 0
			_ = proc
		}
	}
}

// writeState writes the daemon state file.
func (s *Server) writeState(host string, port int) {
	if s.stateFile == "" {
		return
	}
	state := DaemonState{
		Host:  host,
		Port:  port,
		Token: s.token,
		PID:   os.Getpid(),
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	os.MkdirAll(runtime.RunDir(), 0755)
	os.WriteFile(s.stateFile, data, 0644)
}

// removeState removes the daemon state file.
func (s *Server) removeState() {
	if s.stateFile != "" {
		os.Remove(s.stateFile)
	}
}

// log writes a log message.
func (s *Server) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	if s.logFile != "" {
		runtime.WriteLog(s.logFile, line)
	}
}

// sendJSON sends a JSON response.
func (s *Server) sendJSON(conn net.Conn, v interface{}) {
	data, _ := json.Marshal(v)
	conn.Write(data)
	conn.Write([]byte("\n"))
}

// sendError sends an error response.
func (s *Server) sendError(conn net.Conn, msg string) {
	s.sendJSON(conn, map[string]interface{}{"status": "error", "error": msg})
}

// Helper functions for extracting typed values from map
func getStr(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	v, _ := m[key].(bool)
	return v
}
