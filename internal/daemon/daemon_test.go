package daemon

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("new registry count = %d, want 0", r.Count())
	}
}

func TestRegistryNames(t *testing.T) {
	r := NewRegistry()
	names := r.Names()
	if len(names) != 0 {
		t.Errorf("empty registry names = %v, want empty", names)
	}
}

func TestNewWorkerPool(t *testing.T) {
	wp := NewWorkerPool(10)
	if wp == nil {
		t.Fatal("NewWorkerPool returned nil")
	}
	if wp.ActiveWorkers() != 0 {
		t.Errorf("new pool active workers = %d, want 0", wp.ActiveWorkers())
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	wp := NewWorkerPool(10)
	wp.Shutdown() // Should not panic
	if wp.ActiveWorkers() != 0 {
		t.Errorf("after shutdown active workers = %d, want 0", wp.ActiveWorkers())
	}
}
