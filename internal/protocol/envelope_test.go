package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func validEnvelope() *Envelope {
	return &Envelope{
		Version:   ProtocolVersion,
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		TraceID:   "trace-001-abc",
		SessionID: "sess-123",
		Sender:    "sess-123/caller",
		Recipient: "sess-123/worker",
		Type:      FrameRequest,
		Payload:   json.RawMessage(`{"prompt":"hello world"}`),
	}
}

func TestEnvelopeValidation_Success(t *testing.T) {
	env := validEnvelope()
	if err := env.Validate(); err != nil {
		t.Fatalf("expected valid envelope, got error: %v", err)
	}
}

func TestEnvelopeValidation_MissingFields(t *testing.T) {
	cases := []struct {
		name      string
		modify    func(e *Envelope)
		errSubstr string
	}{
		{"missing_version", func(e *Envelope) { e.Version = "" }, "version"},
		{"missing_id", func(e *Envelope) { e.ID = "" }, "id"},
		{"missing_trace_id", func(e *Envelope) { e.TraceID = "" }, "trace_id"},
		{"missing_session_id", func(e *Envelope) { e.SessionID = "" }, "session_id"},
		{"missing_sender", func(e *Envelope) { e.Sender = "" }, "sender"},
		{"missing_recipient", func(e *Envelope) { e.Recipient = "" }, "recipient"},
		{"missing_type", func(e *Envelope) { e.Type = "" }, "type"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validEnvelope()
			tc.modify(env)
			err := env.Validate()
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tc.errSubstr)
			}
			if !strings.Contains(err.Error(), tc.errSubstr) {
				t.Errorf("error %q does not contain expected substring %q", err.Error(), tc.errSubstr)
			}
		})
	}
}

func TestEnvelopeValidation_ReferencesBound(t *testing.T) {
	env := validEnvelope()
	// Exceed MaxReferences (64)
	env.References = make([]FileReference, MaxReferences+1)
	for i := range env.References {
		env.References[i] = FileReference{
			Path:   "/path/to/file",
			SHA256: strings.Repeat("a", 64),
		}
	}

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error on exceeding MaxReferences, got nil")
	}

	// Valid count with invalid SHA256 length
	env.References = []FileReference{
		{Path: "/path/to/file", SHA256: "too_short"},
	}
	if err := env.Validate(); err == nil {
		t.Fatal("expected error on invalid sha256 length, got nil")
	}
}

func TestEnvelope_LosslessExtensions(t *testing.T) {
	rawJSON := `{
		"version": "1.0",
		"id": "test-id",
		"trace_id": "test-trace",
		"session_id": "test-session",
		"sender": "sender-a",
		"recipient": "recipient-b",
		"type": "REQUEST",
		"payload": {"action":"run"},
		"custom_feature_flags": {"gpu_enabled": true},
		"future_spec_annotation": "reserved_value"
	}`

	var env Envelope
	if err := json.Unmarshal([]byte(rawJSON), &env); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if env.Version != "1.0" || env.Sender != "sender-a" {
		t.Errorf("standard fields not parsed correctly: %+v", env)
	}

	if len(env.Extensions) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(env.Extensions))
	}

	if _, ok := env.Extensions["custom_feature_flags"]; !ok {
		t.Error("missing custom_feature_flags in extensions")
	}
	if _, ok := env.Extensions["future_spec_annotation"]; !ok {
		t.Error("missing future_spec_annotation in extensions")
	}

	// Re-marshal to verify roundtrip lossless reproduction
	marshaled, err := json.Marshal(&env)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var roundtrip map[string]interface{}
	if err := json.Unmarshal(marshaled, &roundtrip); err != nil {
		t.Fatalf("unmarshal roundtrip failed: %v", err)
	}

	if roundtrip["future_spec_annotation"] != "reserved_value" {
		t.Errorf("expected extension preserved in roundtrip, got: %v", roundtrip["future_spec_annotation"])
	}
}
