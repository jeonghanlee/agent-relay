package conformance

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jeonghanlee/agent-relay/internal/framing"
	"github.com/jeonghanlee/agent-relay/internal/protocol"
)

func TestWireConformance_AllFrameTypes(t *testing.T) {
	frames := []*protocol.Envelope{
		{
			Version:   protocol.ProtocolVersion,
			ID:        "frame-req-001",
			TraceID:   "trace-req-001",
			SessionID: "sess-main",
			Sender:    "sess-main/user",
			Recipient: "sess-main/agent-claude",
			Type:      protocol.FrameRequest,
			Payload:   json.RawMessage(`{"prompt":"implement layer 0"}`),
			References: []protocol.FileReference{
				{Path: "docs/ARCHITECTURE.md", SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
			},
		},
		{
			Version:   protocol.ProtocolVersion,
			ID:        "frame-prog-002",
			TraceID:   "trace-req-001",
			SessionID: "sess-main",
			Sender:    "sess-main/agent-claude",
			Recipient: "sess-main/user",
			Type:      protocol.FrameEventProgress,
			Payload:   json.RawMessage(`{"phase":"compiling","percent":45}`),
		},
		{
			Version:   protocol.ProtocolVersion,
			ID:        "frame-chunk-003",
			TraceID:   "trace-req-001",
			SessionID: "sess-main",
			Sender:    "sess-main/agent-claude",
			Recipient: "sess-main/user",
			Type:      protocol.FrameEventLogChunk,
			Payload:   json.RawMessage(`{"stream":"stdout","content":"[build] ok\n","seq":1}`),
		},
		{
			Version:   protocol.ProtocolVersion,
			ID:        "frame-resp-004",
			TraceID:   "trace-req-001",
			SessionID: "sess-main",
			Sender:    "sess-main/agent-claude",
			Recipient: "sess-main/user",
			Type:      protocol.FrameResponse,
			Payload:   json.RawMessage(`{"exit_code":0,"summary":"layer 0 verified"}`),
		},
		{
			Version:   protocol.ProtocolVersion,
			ID:        "frame-err-005",
			TraceID:   "trace-req-001",
			SessionID: "sess-main",
			Sender:    "sess-main/agent-claude",
			Recipient: "sess-main/user",
			Type:      protocol.FrameError,
			Payload:   json.RawMessage(`{"code":"ERR_LENGTH_EXCEEDED","message":"payload too large"}`),
		},
	}

	var wire bytes.Buffer

	// Serialize and transmit all frames sequentially across the wire buffer
	for i, orig := range frames {
		if err := orig.Validate(); err != nil {
			t.Fatalf("frame %d failed validation: %v", i, err)
		}

		encodedJSON, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("frame %d marshal failed: %v", i, err)
		}

		if err := framing.WriteFrame(&wire, encodedJSON); err != nil {
			t.Fatalf("frame %d WriteFrame failed: %v", i, err)
		}
	}

	// Read and deserialize all frames sequentially from the wire buffer
	for i, orig := range frames {
		rawPayload, err := framing.ReadFrame(&wire)
		if err != nil {
			t.Fatalf("frame %d ReadFrame failed: %v", i, err)
		}

		var received protocol.Envelope
		if err := json.Unmarshal(rawPayload, &received); err != nil {
			t.Fatalf("frame %d unmarshal failed: %v", i, err)
		}

		if err := received.Validate(); err != nil {
			t.Fatalf("frame %d received invalid envelope: %v", i, err)
		}

		if received.ID != orig.ID || received.Type != orig.Type || received.TraceID != orig.TraceID {
			t.Fatalf("frame %d metadata mismatch: expected %+v, got %+v", i, orig, received)
		}

		if !bytes.Equal(received.Payload, orig.Payload) {
			t.Fatalf("frame %d payload mismatch: expected %s, got %s", i, string(orig.Payload), string(received.Payload))
		}
	}
}

func TestWireConformance_ExtensionLosslessRoundtrip(t *testing.T) {
	orig := &protocol.Envelope{
		Version:   protocol.ProtocolVersion,
		ID:        "frame-ext-001",
		TraceID:   "trace-ext-001",
		SessionID: "sess-ext",
		Sender:    "sender-a",
		Recipient: "recipient-b",
		Type:      protocol.FrameRequest,
		Payload:   json.RawMessage(`{"test":true}`),
		Extensions: map[string]json.RawMessage{
			"custom_gpu_quota": json.RawMessage(`{"vram_mb":8192}`),
			"telemetry_span":   json.RawMessage(`"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"`),
		},
	}

	var wire bytes.Buffer
	encodedJSON, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if err := framing.WriteFrame(&wire, encodedJSON); err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	rawPayload, err := framing.ReadFrame(&wire)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	var received protocol.Envelope
	if err := json.Unmarshal(rawPayload, &received); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(received.Extensions) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(received.Extensions))
	}

	if string(received.Extensions["telemetry_span"]) != `"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"` {
		t.Errorf("extension telemetry_span corrupted: %s", string(received.Extensions["telemetry_span"]))
	}
}
