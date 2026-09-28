package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrMissingRequiredField = errors.New("protocol: missing required field")
	ErrTooManyReferences    = errors.New("protocol: references exceed maximum allowed")
	ErrInvalidSHA256        = errors.New("protocol: reference has invalid sha256 checksum")
)

// Envelope defines the canonical message container transferred across control.sock.
type Envelope struct {
	Version    string                     `json:"version"`
	ID         string                     `json:"id"`
	TraceID    string                     `json:"trace_id"`
	SessionID  string                     `json:"session_id"`
	Sender     string                     `json:"sender"`
	Recipient  string                     `json:"recipient"`
	Type       FrameType                  `json:"type"`
	Payload    json.RawMessage            `json:"payload,omitempty"`
	References []FileReference            `json:"references,omitempty"`
	Extensions map[string]json.RawMessage `json:"-"`
}

type envelopeAlias Envelope

// MarshalJSON preserves standard envelope fields and seamlessly merges Extensions.
func (e *Envelope) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal((*envelopeAlias)(e))
	if err != nil {
		return nil, err
	}

	if len(e.Extensions) == 0 {
		return data, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	for k, v := range e.Extensions {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON extracts known fields and captures unknown fields into Extensions.
func (e *Envelope) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*envelopeAlias)(e)); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	knownKeys := map[string]struct{}{
		"version":    {},
		"id":         {},
		"trace_id":   {},
		"session_id": {},
		"sender":     {},
		"recipient":  {},
		"type":       {},
		"payload":    {},
		"references": {},
	}

	for k := range knownKeys {
		delete(raw, k)
	}

	if len(raw) > 0 {
		e.Extensions = raw
	} else {
		e.Extensions = nil
	}

	return nil
}

// Validate verifies mandatory fields and integrity constraints on the envelope.
func (e *Envelope) Validate() error {
	if e.Version == "" {
		return fmt.Errorf("%w: version", ErrMissingRequiredField)
	}
	if e.ID == "" {
		return fmt.Errorf("%w: id", ErrMissingRequiredField)
	}
	if e.TraceID == "" {
		return fmt.Errorf("%w: trace_id", ErrMissingRequiredField)
	}
	if e.SessionID == "" {
		return fmt.Errorf("%w: session_id", ErrMissingRequiredField)
	}
	if e.Sender == "" {
		return fmt.Errorf("%w: sender", ErrMissingRequiredField)
	}
	if e.Recipient == "" {
		return fmt.Errorf("%w: recipient", ErrMissingRequiredField)
	}
	if e.Type == "" {
		return fmt.Errorf("%w: type", ErrMissingRequiredField)
	}

	if len(e.References) > MaxReferences {
		return fmt.Errorf("%w: got %d, max %d", ErrTooManyReferences, len(e.References), MaxReferences)
	}

	for i, ref := range e.References {
		if ref.Path == "" {
			return fmt.Errorf("%w: reference[%d].path is empty", ErrMissingRequiredField, i)
		}
		if len(ref.SHA256) != 64 {
			return fmt.Errorf("%w: reference[%d].sha256 must be 64 hex characters", ErrInvalidSHA256, i)
		}
	}

	return nil
}
