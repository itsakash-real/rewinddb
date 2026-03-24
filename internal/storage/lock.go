package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const LockFileName = "LOCK"

// ErrLockHeld is returned when a live process already holds the lock.
var ErrLockHeld = errors.New("storage: another rw process is running")

// lockPayload is the JSON written into the LOCK file.
type lockPayload struct {
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
}

// FileLock manages an exclusive .rewind/LOCK file.
type FileLock struct {
	path string
}

// NewFileLock returns a FileLock for the given lock-file path.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Acquire tries to obtain the lock.
//
// If the lock file exists and the recorded PID is alive → ErrLockHeld.
// If the lock file exists and the PID is dead → stale lock, removed and re-acquired.
// If no lock file exists → created immediately.
//
// PID liveness is checked via signal(0) on POSIX [web:112].
func (fl *FileLock) Acquire() error {
	// Try to read an existing lock file.
	if data, err := os.ReadFile(fl.path); err == nil {
		var payload lockPayload
		if jsonErr := json.Unmarshal(data, &payload); jsonErr == nil {
			if isProcessAlive(payload.PID) {
				return fmt.Errorf("%w (PID: %d, since: %s)",
					ErrLockHeld, payload.PID,
					payload.Timestamp.Format(time.RFC3339))
			}
			// Stale lock — process is dead. Remove and reacquire.
			log.Warn().
				Int("stale_pid", payload.PID).
				Str("lock", fl.path).
				Msg("removing stale lock file")
			os.Remove(fl.path)
		}
	}

	payload := lockPayload{
		PID:       os.Getpid(),
		Timestamp: time.Now().UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("storage: marshal lock payload: %w", err)
	}

	// O_EXCL ensures atomic creation — only one process wins [web:40].
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: lock file appeared concurrently", ErrLockHeld)
		}
		return fmt.Errorf("storage: create lock file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(fl.path)
		return fmt.Errorf("storage: write lock file: %w", err)
	}

	log.Debug().Int("pid", os.Getpid()).Str("lock", fl.path).Msg("lock acquired")
	return nil
}

// Release removes the LOCK file. It is safe to call multiple times.
func (fl *FileLock) Release() error {
	if err := os.Remove(fl.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: release lock: %w", err)
	}
	log.Debug().Str("lock", fl.path).Msg("lock released")
	return nil
}

// WithLock acquires the lock, runs fn, then releases regardless of fn's error.
func (fl *FileLock) WithLock(fn func() error) error {
	if err := fl.Acquire(); err != nil {
		return err
	}
	defer fl.Release()
	return fn()
}

// isProcessAlive returns true if pid corresponds to a live process.
// Uses signal(0) on POSIX — a no-op signal that only tests the PID [web:112].
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On POSIX, FindProcess always succeeds; signal(0) is the correct liveness probe [web:109].
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// ESRCH = no such process; EPERM = process exists but we lack permission.
	if errors.Is(err, syscall.EPERM) {
		return true // process alive, we just can't signal it
	}
	return false // ESRCH or other error → process is dead
}
