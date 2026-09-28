package framing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	// MaxFrameSize defines the hard 16MB upper bound on single frame payloads.
	MaxFrameSize = 16 * 1024 * 1024

	// DefaultHeaderTimeout defines the maximum time allowed to read the 4-byte header.
	DefaultHeaderTimeout = 10 * time.Second
)

var (
	ErrFrameTooLarge = fmt.Errorf("framing: frame length exceeds maximum allowed limit of %d bytes", MaxFrameSize)
	ErrEmptyFrame    = errors.New("framing: frame payload is empty")
)

// WriteFrame transmits a 4-byte big-endian length prefix followed by the payload.
func WriteFrame(w io.Writer, payload []byte) error {
	payloadLen := len(payload)
	if payloadLen > MaxFrameSize {
		return ErrFrameTooLarge
	}

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(payloadLen))

	// Write length prefix
	if _, err := w.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("framing: failed to write length prefix: %w", err)
	}

	// Write payload if non-empty
	if payloadLen > 0 {
		if _, err := w.Write(payload); err != nil {
			return fmt.Errorf("framing: failed to write frame payload: %w", err)
		}
	}

	return nil
}

// ReadFrame reads a 4-byte length prefix and returns the exact payload slice.
// If the length prefix exceeds MaxFrameSize, it rejects the frame immediately
// without allocating the payload buffer, protecting the daemon from memory-exhaustion DoS.
func ReadFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	if length == 0 {
		return []byte{}, nil
	}

	payload := make([]byte, length)
	// Use io.LimitReader to enforce length bound strictly
	limitedReader := io.LimitReader(r, int64(length))
	if _, err := io.ReadFull(limitedReader, payload); err != nil {
		return nil, fmt.Errorf("framing: failed to read complete payload (%d bytes expected): %w", length, err)
	}

	return payload, nil
}

// ReadFrameWithDeadline wraps ReadFrame with an explicit connection deadline on net.Conn.
func ReadFrameWithDeadline(conn net.Conn, timeout time.Duration) ([]byte, error) {
	if timeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return nil, fmt.Errorf("framing: failed to set read deadline: %w", err)
		}
		defer conn.SetReadDeadline(time.Time{})
	}
	return ReadFrame(conn)
}
