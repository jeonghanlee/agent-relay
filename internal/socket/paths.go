package socket

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultSocketName = "control.sock"
	DefaultLockName   = "agent-relay.lock"
)

// ResolveRuntimeDir determines the secure runtime directory for sockets and locks.
// Priority:
// 1. $XDG_RUNTIME_DIR/agent-relay (mode 0700)
// 2. /tmp/agent-relay-<UID> (mode 0700) fallback
func ResolveRuntimeDir() (string, error) {
	uid := os.Getuid()

	var candidate string
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidate = filepath.Join(xdg, "agent-relay")
	} else {
		candidate = filepath.Join(os.TempDir(), fmt.Sprintf("agent-relay-%d", uid))
	}

	// Ensure directory exists with strict 0700 permissions
	if err := os.MkdirAll(candidate, 0700); err != nil {
		return "", fmt.Errorf("socket: failed to create runtime directory %q: %w", candidate, err)
	}

	// Verify permissions strictly match 0700 (owner only)
	info, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("socket: failed to stat runtime directory: %w", err)
	}

	perm := info.Mode().Perm()
	if perm != 0700 {
		if err := os.Chmod(candidate, 0700); err != nil {
			return "", fmt.Errorf("socket: failed to enforce 0700 permissions on %q: %w", candidate, err)
		}
	}

	return candidate, nil
}

// ResolveSocketPath returns the full canonical path for control.sock.
func ResolveSocketPath() (string, error) {
	dir, err := ResolveRuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultSocketName), nil
}

// ResolveLockPath returns the full canonical path for agent-relay.lock.
func ResolveLockPath() (string, error) {
	dir, err := ResolveRuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultLockName), nil
}
