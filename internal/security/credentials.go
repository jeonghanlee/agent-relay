package security

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

var (
	ErrUnauthorizedUID = errors.New("security: caller UID does not match daemon UID")
	ErrInvalidConn     = errors.New("security: connection is not a valid unix domain socket")
)

// PeerCredentials encapsulates caller identity information extracted from the kernel.
type PeerCredentials struct {
	PID uint32
	UID uint32
	GID uint32
}

// VerifyPeerCredentials extracts the SO_PEERCRED structure from a UnixConn
// and asserts that the caller's UID strictly matches expectedUID.
// If expectedUID is 0, the current process UID (os.Getuid()) is used.
func VerifyPeerCredentials(conn *net.UnixConn, expectedUID uint32) (*PeerCredentials, error) {
	if conn == nil {
		return nil, ErrInvalidConn
	}

	if expectedUID == 0 {
		expectedUID = uint32(os.Getuid())
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("security: failed to acquire raw connection: %w", err)
	}

	var ucred *unix.Ucred
	var sockErr error

	controlErr := rawConn.Control(func(fd uintptr) {
		ucred, sockErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})

	if controlErr != nil {
		return nil, fmt.Errorf("security: raw connection control failed: %w", controlErr)
	}
	if sockErr != nil {
		return nil, fmt.Errorf("security: failed to query SO_PEERCRED: %w", sockErr)
	}
	if ucred == nil {
		return nil, errors.New("security: kernel returned nil credentials")
	}

	if ucred.Uid != expectedUID {
		return nil, fmt.Errorf("%w: caller UID %d != daemon UID %d", ErrUnauthorizedUID, ucred.Uid, expectedUID)
	}

	return &PeerCredentials{
		PID: uint32(ucred.Pid),
		UID: ucred.Uid,
		GID: ucred.Gid,
	}, nil
}
