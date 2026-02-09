package lock

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProviderLock provides per-provider, per-directory file locking to serialize request-response cycles.
// Lock files are stored in ~/.ccb/run/{provider}-{cwd_hash}.lock
type ProviderLock struct {
	Provider string
	Timeout  time.Duration
	LockDir  string
	LockFile string
	fd       *os.File
	acquired bool
}

// NewProviderLock creates a new lock for a specific provider and working directory.
func NewProviderLock(provider string, timeout time.Duration, cwd string) *ProviderLock {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	home, _ := os.UserHomeDir()
	lockDir := filepath.Join(home, ".ccb", "run")

	hash := md5.Sum([]byte(cwd))
	cwdHash := fmt.Sprintf("%x", hash)[:8]
	lockFile := filepath.Join(lockDir, fmt.Sprintf("%s-%s.lock", provider, cwdHash))

	return &ProviderLock{
		Provider: provider,
		Timeout:  timeout,
		LockDir:  lockDir,
		LockFile: lockFile,
	}
}

// isPIDAlive checks if a process with the given PID is still running.
func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. We need to send signal 0 to check.
	// On Windows, FindProcess only succeeds if the process exists.
	return checkProcessAlive(proc)
}

// TryAcquire attempts to acquire the lock without blocking.
func (l *ProviderLock) TryAcquire() bool {
	os.MkdirAll(l.LockDir, 0755)

	f, err := os.OpenFile(l.LockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return false
	}
	l.fd = f

	if l.tryLockOnce() {
		return true
	}

	// Check for stale lock
	if l.checkStaleLock() {
		l.fd.Close()
		f, err = os.OpenFile(l.LockFile, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return false
		}
		l.fd = f
		if l.tryLockOnce() {
			return true
		}
	}

	l.fd.Close()
	l.fd = nil
	return false
}

// Acquire acquires the lock, waiting up to Timeout.
func (l *ProviderLock) Acquire() bool {
	os.MkdirAll(l.LockDir, 0755)

	f, err := os.OpenFile(l.LockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return false
	}
	l.fd = f

	deadline := time.Now().Add(l.Timeout)
	staleChecked := false

	for time.Now().Before(deadline) {
		if l.tryLockOnce() {
			return true
		}

		if !staleChecked {
			staleChecked = true
			if l.checkStaleLock() {
				l.fd.Close()
				f, err = os.OpenFile(l.LockFile, os.O_CREATE|os.O_RDWR, 0600)
				if err != nil {
					return false
				}
				l.fd = f
				if l.tryLockOnce() {
					return true
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	if l.fd != nil {
		l.fd.Close()
		l.fd = nil
	}
	return false
}

// Release releases the lock.
func (l *ProviderLock) Release() {
	if l.fd != nil {
		if l.acquired {
			unlockFile(l.fd)
		}
		l.fd.Close()
		l.fd = nil
		l.acquired = false
	}
}

// tryLockOnce attempts to acquire the file lock once.
func (l *ProviderLock) tryLockOnce() bool {
	if err := lockFile(l.fd); err != nil {
		return false
	}

	// Write PID
	pid := fmt.Sprintf("%d\n", os.Getpid())
	l.fd.Seek(0, 0)
	l.fd.WriteString(pid)
	l.fd.Truncate(int64(len(pid)))
	l.acquired = true
	return true
}

// checkStaleLock checks if the current lock holder is dead.
func (l *ProviderLock) checkStaleLock() bool {
	data, err := os.ReadFile(l.LockFile)
	if err != nil {
		return false
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return false
	}
	pid, err := strconv.Atoi(content)
	if err != nil {
		return false
	}
	if !isPIDAlive(pid) {
		os.Remove(l.LockFile)
		return true
	}
	return false
}
