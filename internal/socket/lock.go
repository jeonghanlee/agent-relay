package socket

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	ErrDaemonAlreadyRunning = errors.New("socket: another instance of agent-relay daemon is already running")
)

// InstanceLock guarantees single-instance execution via Linux kernel flock.
type InstanceLock struct {
	file *os.File
	path string
}

// AcquireInstanceLock obtains an exclusive non-blocking file lock on lockPath.
// If the lock is currently held by another process, it returns ErrDaemonAlreadyRunning.
func AcquireInstanceLock(lockPath string) (*InstanceLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("socket: failed to open lockfile %q: %w", lockPath, err)
	}

	// Apply exclusive, non-blocking lock
	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			// Read existing PID if available for diagnostics
			existingPID, _ := os.ReadFile(lockPath)
			pidStr := strings.TrimSpace(string(existingPID))
			return nil, fmt.Errorf("%w (PID %s holds lock %s)", ErrDaemonAlreadyRunning, pidStr, lockPath)
		}
		return nil, fmt.Errorf("socket: failed to acquire flock on %q: %w", lockPath, err)
	}

	// Record current process PID into lockfile
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	_ = f.Sync()

	return &InstanceLock{
		file: f,
		path: lockPath,
	}, nil
}

// Release unlocks the file descriptor and removes the lock file.
func (l *InstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	var flockErr error
	if err := unix.Flock(int(l.file.Fd()), unix.LOCK_UN); err != nil {
		flockErr = fmt.Errorf("socket: failed to release flock: %w", err)
	}

	closeErr := l.file.Close()
	removeErr := os.Remove(l.path)

	if flockErr != nil {
		return flockErr
	}
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}

	return nil
}
