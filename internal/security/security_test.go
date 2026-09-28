package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyPeerCredentials_RealSocket(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "test.sock")

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sockPath, Net: "unix"})
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	defer l.Close()

	credCh := make(chan *PeerCredentials, 1)
	errCh := make(chan error, 1)

	go func() {
		conn, err := l.AcceptUnix()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		// Verify against current process UID
		creds, err := VerifyPeerCredentials(conn, uint32(os.Getuid()))
		if err != nil {
			errCh <- err
			return
		}
		credCh <- creds
	}()

	client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: sockPath, Net: "unix"})
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer client.Close()

	select {
	case creds := <-credCh:
		if creds.UID != uint32(os.Getuid()) {
			t.Errorf("expected UID %d, got %d", os.Getuid(), creds.UID)
		}
		if creds.PID == 0 {
			t.Error("expected non-zero PID from kernel")
		}
	case err := <-errCh:
		t.Fatalf("credential verification failed: %v", err)
	}
}

func TestVerifyPeerCredentials_UnauthorizedUID(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "test-unauth.sock")

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sockPath, Net: "unix"})
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	errCh := make(chan error, 1)

	go func() {
		conn, err := l.AcceptUnix()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		// Demand a different UID that does not match current caller
		bogusUID := uint32(os.Getuid() + 9999)
		_, verifyErr := VerifyPeerCredentials(conn, bogusUID)
		errCh <- verifyErr
	}()

	client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: sockPath, Net: "unix"})
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer client.Close()

	err = <-errCh
	if !errors.Is(err, ErrUnauthorizedUID) {
		t.Fatalf("expected ErrUnauthorizedUID, got %v", err)
	}
}

func TestValidateWorkspacePath(t *testing.T) {
	tempDir := t.TempDir()
	workspaceRoot := filepath.Join(tempDir, "workspace")
	outsideDir := filepath.Join(tempDir, "outside")

	if err := os.MkdirAll(workspaceRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatal(err)
	}

	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("classified"), 0600); err != nil {
		t.Fatal(err)
	}

	validFile := filepath.Join(workspaceRoot, "safe.txt")
	if err := os.WriteFile(validFile, []byte("safe"), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Valid relative file inside workspace
	res, err := ValidateWorkspacePath(workspaceRoot, "safe.txt")
	if err != nil {
		t.Fatalf("expected safe.txt to validate, got %v", err)
	}
	if res != validFile {
		t.Errorf("expected %s, got %s", validFile, res)
	}

	// 2. Traversal attempt via ".."
	_, err = ValidateWorkspacePath(workspaceRoot, "../outside/secret.txt")
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal on ../ outside, got %v", err)
	}

	// 3. Symlink pointing outside workspace
	symlinkPath := filepath.Join(workspaceRoot, "leak-link")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	_, err = ValidateWorkspacePath(workspaceRoot, "leak-link")
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal on escaping symlink, got %v", err)
	}

	// 4. Non-existent write target inside workspace
	newTarget := filepath.Join(workspaceRoot, "subdir", "output.log")
	res, err = ValidateWorkspacePath(workspaceRoot, "subdir/output.log")
	if err != nil {
		t.Fatalf("expected valid non-existent target inside workspace, got %v", err)
	}
	if res != newTarget {
		t.Errorf("expected %s, got %s", newTarget, res)
	}
}

func TestVerifyFileSHA256(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "sample.txt")
	content := []byte("accelerator controls data stream")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256(content)
	expectedHex := hex.EncodeToString(hash[:])

	// Matching hash
	if err := VerifyFileSHA256(testFile, expectedHex); err != nil {
		t.Fatalf("expected hash to verify, got error: %v", err)
	}

	// Case insensitive
	if err := VerifyFileSHA256(testFile, strings.ToUpper(expectedHex)); err != nil {
		t.Fatalf("expected uppercase hash to verify, got error: %v", err)
	}

	// Mismatched hash
	bogusHex := strings.Repeat("0", 64)
	if err := VerifyFileSHA256(testFile, bogusHex); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}

	// Invalid length
	if err := VerifyFileSHA256(testFile, "abc123"); !errors.Is(err, ErrInvalidChecksum) {
		t.Fatalf("expected ErrInvalidChecksum on short hash, got %v", err)
	}
}

func TestSanitizeEnvironment(t *testing.T) {
	raw := []string{
		"USER=operator",
		"LD_PRELOAD=/tmp/evil.so",
		"IFS=:",
		"BASH_ENV=/tmp/pwn.bash",
		"HOME=/home/operator",
		"DYLD_INSERT_LIBRARIES=/tmp/mac_evil.dylib",
	}

	clean := SanitizeEnvironment(raw)

	for _, entry := range clean {
		for blocked := range DangerousEnvVars {
			if strings.HasPrefix(entry, blocked+"=") {
				t.Errorf("dangerous env var %q leaked into sanitized environment", entry)
			}
		}
	}

	// Assert PATH was injected
	hasPath := false
	for _, entry := range clean {
		if strings.HasPrefix(entry, "PATH=") {
			hasPath = true
			break
		}
	}
	if !hasPath {
		t.Error("expected PATH to be injected into sanitized environment")
	}
}
