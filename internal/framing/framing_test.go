package framing

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestFramingRoundtrip_Success(t *testing.T) {
	testCases := [][]byte{
		[]byte("hello world"),
		[]byte(""),
		bytes.Repeat([]byte("A"), 1024),         // 1KB
		bytes.Repeat([]byte("B"), 1024*1024),    // 1MB
		bytes.Repeat([]byte("C"), MaxFrameSize), // Exact 16MB boundary
	}

	for i, tc := range testCases {
		var buf bytes.Buffer
		if err := WriteFrame(&buf, tc); err != nil {
			t.Fatalf("case %d: WriteFrame failed: %v", i, err)
		}

		payload, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("case %d: ReadFrame failed: %v", i, err)
		}

		if !bytes.Equal(payload, tc) {
			t.Fatalf("case %d: payload mismatch (len expected %d, got %d)", i, len(tc), len(payload))
		}
	}
}

func TestWriteFrame_RejectsExceedingMax(t *testing.T) {
	oversized := make([]byte, MaxFrameSize+1)
	var buf bytes.Buffer
	err := WriteFrame(&buf, oversized)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
}

func TestReadFrame_RejectsExceedingMax_DoSProtection(t *testing.T) {
	var buf bytes.Buffer
	// Craft an adversarial 4-byte header claiming a 100MB payload
	var fakeHeader [4]byte
	binary.BigEndian.PutUint32(fakeHeader[:], 100*1024*1024)
	buf.Write(fakeHeader[:])

	// Attempt reading from crafted stream
	payload, err := ReadFrame(&buf)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
	if payload != nil {
		t.Fatalf("expected nil payload on rejection, got %v", payload)
	}
}

func TestReadFrame_TruncatedPayload(t *testing.T) {
	var buf bytes.Buffer
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], 100) // Expects 100 bytes
	buf.Write(lenBuf[:])
	buf.Write([]byte("short")) // Only 5 bytes provided

	_, err := ReadFrame(&buf)
	if err == nil {
		t.Fatal("expected error on truncated stream, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read complete payload") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadFrame_EOFHeader(t *testing.T) {
	var buf bytes.Buffer
	_, err := ReadFrame(&buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF on empty reader, got %v", err)
	}
}

func TestReadFrameWithDeadline_Timeout(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		// Server attempts to read with 50ms deadline, but client never writes
		_, err := ReadFrameWithDeadline(server, 50*time.Millisecond)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected read timeout error, got nil")
		}
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Logf("observed timeout error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for read deadline to fire")
	}
}
