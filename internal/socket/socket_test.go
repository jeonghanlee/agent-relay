package socket

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestResolveRuntimeDir(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tempDir)

	dir, err := ResolveRuntimeDir()
	if err != nil {
		t.Fatalf("failed to resolve runtime dir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat runtime dir: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("expected permissions 0700, got %04o", perm)
	}

	sockPath, err := ResolveSocketPath()
	if err != nil {
		t.Fatalf("failed to resolve socket path: %v", err)
	}
	if filepath.Base(sockPath) != DefaultSocketName {
		t.Errorf("expected base %s, got %s", DefaultSocketName, filepath.Base(sockPath))
	}
}

func TestInstanceLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "daemon.lock")

	lock1, err := AcquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	// Second acquire must fail with ErrDaemonAlreadyRunning
	_, err = AcquireInstanceLock(lockPath)
	if !errors.Is(err, ErrDaemonAlreadyRunning) {
		t.Fatalf("expected ErrDaemonAlreadyRunning, got %v", err)
	}

	// Release first lock
	if err := lock1.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}

	// Third acquire should now succeed
	lock2, err := AcquireInstanceLock(lockPath)
	if err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
	defer lock2.Release()
}

func TestListener_LifecycleAndPermissions(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "control.sock")
	lockPath := filepath.Join(tempDir, "control.lock")

	l, err := Listen(sockPath, lockPath)
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer l.Close()

	// Check socket permissions are 0600
	info, err := os.Stat(sockPath)
	if err != nil {
		t.Fatalf("stat socket failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected socket permissions 0600, got %04o", perm)
	}

	// Verify client dial and accept
	clientConn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer clientConn.Close()

	serverConn, err := l.AcceptUnix()
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	defer serverConn.Close()

	// Concurrent listen on same path must fail
	_, err = Listen(sockPath, lockPath)
	if !errors.Is(err, ErrDaemonAlreadyRunning) {
		t.Fatalf("expected ErrDaemonAlreadyRunning on concurrent listen, got %v", err)
	}

	// Close listener and verify cleanup
	if err := l.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Errorf("expected socket file to be removed after close")
	}
}

func TestListener_StaleSocketRecovery(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "stale.sock")
	lockPath := filepath.Join(tempDir, "stale.lock")

	// Create a real kernel-level stale socket file simulating a hard crash (SIGKILL):
	// Bind socket to filesystem and close fd WITHOUT unlinking the socket file.
	rawFd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("failed to create unix socket: %v", err)
	}
	addr := &unix.SockaddrUnix{Name: sockPath}
	if err := unix.Bind(rawFd, addr); err != nil {
		unix.Close(rawFd)
		t.Fatalf("failed to bind raw unix socket: %v", err)
	}
	_ = unix.Close(rawFd)

	// Verify the stale socket file exists on disk
	if _, err := os.Stat(sockPath); err != nil {
		t.Fatalf("expected stale socket file to remain on disk: %v", err)
	}

	// Listen should detect the dead socket (ECONNREFUSED on dial probe), remove it, and bind cleanly
	l, err := Listen(sockPath, lockPath)
	if err != nil {
		t.Fatalf("expected successful recovery of stale socket, got: %v", err)
	}
	defer l.Close()

	// Verify new listener is functional
	c, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial on recovered socket failed: %v", err)
	}
	c.Close()
}
