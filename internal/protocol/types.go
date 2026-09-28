package protocol

// ProtocolVersion defines the current wire protocol version.
const ProtocolVersion = "1.0"

// MaxReferences defines the hard upper bound on file references attached to a single envelope.
const MaxReferences = 64

// FrameType identifies the semantic purpose of a frame.
type FrameType string

const (
	FrameRequest       FrameType = "REQUEST"
	FrameEventProgress FrameType = "EVENT_PROGRESS"
	FrameEventLogChunk FrameType = "EVENT_LOG_CHUNK"
	FrameResponse      FrameType = "RESPONSE"
	FrameError         FrameType = "ERROR"
)

// ErrorCode categorizes errors in ERROR frames or status payloads.
type ErrorCode string

const (
	ErrCodeInvalidFrame        ErrorCode = "ERR_INVALID_FRAME"
	ErrCodeLengthExceeded      ErrorCode = "ERR_LENGTH_EXCEEDED"
	ErrCodeUnauthorized        ErrorCode = "ERR_UNAUTHORIZED"
	ErrCodeTimeout             ErrorCode = "ERR_TIMEOUT"
	ErrCodeVersionIncompatible ErrorCode = "ERR_VERSION_INCOMPATIBLE"
	ErrCodePathTraversal       ErrorCode = "ERR_PATH_TRAVERSAL"
	ErrCodeExecutionFailed     ErrorCode = "ERR_EXECUTION_FAILED"
	ErrCodeInternal            ErrorCode = "ERR_INTERNAL"
)

// FileReference represents a pre-validated workspace file attachment.
type FileReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ErrorPayload represents the standard body for ERROR frames.
type ErrorPayload struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

// ResponsePayload represents the standard terminal execution result.
type ResponsePayload struct {
	ExitCode   int    `json:"exit_code"`
	Summary    string `json:"summary,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// LogChunkPayload represents an incremental stdout or stderr stream chunk.
type LogChunkPayload struct {
	Stream   string `json:"stream"`
	Content  string `json:"content"`
	Sequence uint64 `json:"seq"`
}
