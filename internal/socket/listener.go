package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// Listener wraps a net.Listener with UDS lifecycle cleanup and instance lock management.
type Listener struct {
	net.Listener
	socketPath  string
	lock        *InstanceLock
	fromSystemd bool
}

// SocketPath returns the bound socket path, or empty if inherited from systemd.
func (l *Listener) SocketPath() string {
	return l.socketPath
}

// FromSystemd returns true if the listener was inherited via LISTEN_FDS socket activation.
func (l *Listener) FromSystemd() bool {
	return l.fromSystemd
}

// Listen creates or inherits a Unix domain socket listener for agent-relay.
// If socketPath or lockPath are empty, secure defaults from ResolveRuntimeDir are used.
func Listen(socketPath string, lockPath string) (*Listener, error) {
	// 1. Detect systemd socket activation (LISTEN_FDS=1)
	listenFds := os.Getenv("LISTEN_FDS")
	listenPid := os.Getenv("LISTEN_PID")

	if listenFds == "1" && (listenPid == "" || listenPid == strconv.Itoa(os.Getpid())) {
		// File descriptor 3 is standard for SD_LISTEN_FDS_START
		file := os.NewFile(uintptr(3), "systemd-socket")
		if file == nil {
			return nil, errors.New("socket: failed to open systemd socket descriptor 3")
		}
		defer file.Close()

		l, err := net.FileListener(file)
		if err != nil {
			return nil, fmt.Errorf("socket: failed to create listener from systemd socket: %w", err)
		}

		return &Listener{
			Listener:    l,
			fromSystemd: true,
		}, nil
	}

	// 2. Standalone execution: resolve paths if not explicitly provided
	if socketPath == "" {
		resolved, err := ResolveSocketPath()
		if err != nil {
			return nil, err
		}
		socketPath = resolved
	}

	if lockPath == "" {
		resolved, err := ResolveLockPath()
		if err != nil {
			return nil, err
		}
		lockPath = resolved
	}

	// 3. Acquire exclusive flock to prevent multiple daemons
	lock, err := AcquireInstanceLock(lockPath)
	if err != nil {
		return nil, err
	}

	// 4. Stale socket detection and recovery
	if _, err := os.Stat(socketPath); err == nil {
		// Probe if another daemon is actively answering on the socket
		conn, dialErr := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
		if dialErr == nil {
			// Socket is alive and responding
			conn.Close()
			_ = lock.Release()
			return nil, fmt.Errorf("%w: socket %q is active", ErrDaemonAlreadyRunning, socketPath)
		}

		// Socket file exists but connection was refused (dead/stale socket from previous crash)
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			_ = lock.Release()
			return nil, fmt.Errorf("socket: failed to remove stale socket %q: %w", socketPath, err)
		}
	}

	// 5. Bind listener
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = lock.Release()
		return nil, fmt.Errorf("socket: failed to listen on %q: %w", socketPath, err)
	}

	// Enforce 0600 on socket file so only the owning user can connect
	if err := os.Chmod(socketPath, 0600); err != nil {
		_ = l.Close()
		_ = lock.Release()
		_ = os.Remove(socketPath)
		return nil, fmt.Errorf("socket: failed to set 0600 permissions on %q: %w", socketPath, err)
	}

	return &Listener{
		Listener:    l,
		socketPath:  socketPath,
		lock:        lock,
		fromSystemd: false,
	}, nil
}

// AcceptUnix accepts an incoming connection and casts to *net.UnixConn.
func (l *Listener) AcceptUnix() (*net.UnixConn, error) {
	if unixL, ok := l.Listener.(*net.UnixListener); ok {
		return unixL.AcceptUnix()
	}

	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		conn.Close()
		return nil, errors.New("socket: accepted connection is not *net.UnixConn")
	}

	return unixConn, nil
}

// Close gracefully closes the socket listener and releases filesystem locks.
func (l *Listener) Close() error {
	if l == nil {
		return nil
	}

	var closeErr error
	if l.Listener != nil {
		closeErr = l.Listener.Close()
	}

	if !l.fromSystemd && l.socketPath != "" {
		_ = os.Remove(l.socketPath)
	}

	if l.lock != nil {
		_ = l.lock.Release()
	}

	return closeErr
}
