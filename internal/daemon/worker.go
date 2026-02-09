package daemon

import (
	"context"
	"sync"

	"github.com/anthropics/claude_code_bridge/internal/daemon/adapter"
)

// WorkerPool manages per-session goroutine workers for processing requests.
type WorkerPool struct {
	mu      sync.Mutex
	workers map[string]*sessionWorker
	maxSize int
}

type sessionWorker struct {
	sessionKey string
	taskCh     chan *adapter.QueuedTask
	cancel     context.CancelFunc
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(maxSize int) *WorkerPool {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &WorkerPool{
		workers: make(map[string]*sessionWorker),
		maxSize: maxSize,
	}
}

// Submit submits a task to the worker for the given session key.
// If no worker exists for the session, one is created.
func (p *WorkerPool) Submit(sessionKey string, task *adapter.QueuedTask, handler func(context.Context, *adapter.QueuedTask)) {
	p.mu.Lock()
	w, ok := p.workers[sessionKey]
	if !ok {
		ctx, cancel := context.WithCancel(context.Background())
		w = &sessionWorker{
			sessionKey: sessionKey,
			taskCh:     make(chan *adapter.QueuedTask, 16),
			cancel:     cancel,
		}
		p.workers[sessionKey] = w
		go p.runWorker(ctx, w, handler)
	}
	p.mu.Unlock()

	// Non-blocking send; if channel is full, run in a new goroutine
	select {
	case w.taskCh <- task:
	default:
		go handler(task.Ctx, task)
	}
}

// runWorker processes tasks for a single session.
func (p *WorkerPool) runWorker(ctx context.Context, w *sessionWorker, handler func(context.Context, *adapter.QueuedTask)) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-w.taskCh:
			if !ok {
				return
			}
			handler(task.Ctx, task)
		}
	}
}

// Shutdown stops all workers.
func (p *WorkerPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, w := range p.workers {
		w.cancel()
		close(w.taskCh)
		delete(p.workers, key)
	}
}

// ActiveWorkers returns the number of active session workers.
func (p *WorkerPool) ActiveWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.workers)
}
