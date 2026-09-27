# agent-relay Architecture Specification

## Scope

This document specifies the technical architecture, security boundaries, process lifecycle invariants, and wire protocols of `agent-relay`. It serves as the authoritative engineering reference for all daemon and client components.

**Out of scope:** Installation packaging on third-party distributions, remote cloud proxy bridges, and end-user model prompt engineering.

## 1. System Topology and Core Separation

`agent-relay` separates message routing and subprocess execution from high-level agent logic:

- **Daemon (`agent-relay daemon`):** A long-running Unix daemon listening on a Unix-domain socket (`control.sock`). It manages child process lifecycles, enforces kernel security bounds, and routes streaming messages.
- **Client (`agent-relay ctl`):** A lightweight CLI tool invoked by scripts, operators, or peer agents to submit requests, stream execution events, and query daemon status.
- **Storage and Sockets:**
  - Socket path: `$XDG_RUNTIME_DIR/agent-relay/control.sock` (with fallback to `/tmp/agent-relay-${UID}/control.sock`).
  - Lockfile: `control.sock.lock` protected by `unix.Flock`.
  - State directory: `$XDG_STATE_HOME/agent-relay/` (holding post-mortem crash dumps under `crashes/<trace_id>.json`).

## 2. Wire Framing and Protocol (Layer 0)

All communication over `control.sock` uses binary length-prefixed framing:

- **Length Prefix:** 4-byte big-endian `uint32` encoding the byte length of the following payload.
- **Maximum Frame Size:** 16MB (`16 * 1024 * 1024` bytes). Any frame header advertising a length greater than 16MB is rejected immediately prior to buffer allocation to eliminate memory-exhaustion DoS attacks.
- **Read Deadline:** Fixed 10-second read deadline per frame header to prevent Slowloris attacks.
- **Envelope Schema:**
  - `version`: Protocol version string (initially `"1.0"`). Major version mismatches return `ERR_VERSION_INCOMPATIBLE`.
  - `id`: Unique frame identifier (UUIDv4).
  - `trace_id`: Distributed transaction tracing identifier propagated across envelopes and log lines.
  - `session_id`: Unique session scope identifier.
  - `sender`: Identifier of the sending entity (`<session_id>/<agent>`).
  - `recipient`: Target entity identifier.
  - `type`: Frame type (`REQUEST`, `EVENT_PROGRESS`, `EVENT_LOG_CHUNK`, `RESPONSE`, `ERROR`).
  - `payload`: JSON-encoded message body.
  - `references`: Array of up to 64 file references (`MaxReferences = 64`), each containing `path` and pre-validated `sha256`.
  - `extensions`: `map[string]json.RawMessage` preserving unknown fields for lossless forward compatibility.

## 3. Kernel Security and Host Protection (Layer 1)

`agent-relay` enforces kernel-level security guarantees on every connection and filesystem access:

- **Caller Authentication (`SO_PEERCRED`):** The daemon extracts caller credentials via `unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)` on `Accept()`. Any connection originating from a UID differing from the daemon UID is rejected with `ERR_UNAUTHORIZED`.
- **Path Traversal Sandboxing:** All file paths received in `references` or command arguments are resolved through `filepath.EvalSymlinks`. Any path escaping authorized project boundaries is rejected with `ERR_PATH_TRAVERSAL`.
- **TOCTOU Race Prevention:** Files referenced in envelopes have their SHA-256 hashes computed immediately before subprocess dispatch and compared against the envelope hash.
- **Environment Sanitization:** Child processes receive an explicitly allowlisted environment. Dangerous variables (`LD_PRELOAD`, `LD_LIBRARY_PATH`, `BASH_ENV`, `IFS`) are stripped unconditionally.
- **Isolated TMPDIR:** Each execution session receives a private temporary directory with `0700` permissions cleaned up upon session termination.

## 4. Subprocess Lifecycle and Resource Quotas (Layer 2)

Child agents are spawned with strict operating system boundaries:

- **Subreaper Registration (`PR_SET_CHILD_SUBREAPER`):** The daemon registers itself as a subreaper (`unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, ...)`). Grandchild processes reparent to the relay daemon upon parent death, preventing zombie accumulation.
- **Orphan Prevention (`PR_SET_PDEATHSIG`):** Every spawned subprocess executes `unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(unix.SIGTERM), ...)` in `syscall.SysProcAttr.Pdeathsig`. When the daemon terminates, all child agents are signaled immediately.
- **Core Dump Protection (`PR_SET_DUMPABLE = 0`):** Subprocesses disable core dumping to prevent leaking authentication tokens or prompt data.
- **Hardware Resource Limits (`unix.Setrlimit`):**
  - Virtual Memory (`RLIMIT_AS`): Capped at 4GB (`4294967296` bytes).
  - File Size (`RLIMIT_FSIZE`): Capped at 100MB (`104857600` bytes).
  - Open Files (`RLIMIT_NOFILE`): Capped at 1024 descriptors.
- **Kernel OOM Demotion:** Child processes set `/proc/<pid>/oom_score_adj` to `+500`, ensuring the kernel targets rogue child agents rather than critical system daemons during memory pressure.
- **File Descriptor Leak Prevention:** All sockets, pipes, and file descriptors are opened with `FD_CLOEXEC`. Descriptor leakage is verified via `/proc/<pid>/fd/` assertion tests.

## 5. Single-Turn Synchronous Streaming

Each interaction follows a strict synchronous request-event-response lifecycle:

1. **Connect:** Client establishes a UDS connection to `control.sock`.
2. **Handshake:** Protocol version `"1.0"` is confirmed.
3. **Request:** Client transmits a `REQUEST` frame with a unique `trace_id`.
4. **Streaming:** Daemon streams `EVENT_PROGRESS` and `EVENT_LOG_CHUNK` frames back to the client. Log chunks are coalesced up to 64KB or 50ms intervals.
5. **Response:** Execution terminates with a final `RESPONSE` frame carrying exit code and output summary, followed by immediate connection closure.
6. **Cancellation Cascade:** If the client disconnects prematurely, the daemon context cancels immediately, issuing `SIGTERM` (and `SIGKILL` after 500ms) to the corresponding child process group.

## 6. CLI Suite and Operations (Layer 3)

The single binary `agent-relay` multiplexes all functionality:

- `agent-relay daemon`: Runs the foreground or systemd-activated daemon.
- `agent-relay ctl send <agent> "<message>"`: One-line synchronous execution wrapper.
- `agent-relay ctl status --json`: Real-time daemon introspection (active sessions, memory, PIDs).
- `agent-relay ctl doctor`: Preflight diagnostic checking socket status, permissions, and templates.
- `agent-relay ctl abort --all`: Emergency global scram switch terminating all process groups within 500ms.
- `agent-relay ctl prune --days 7`: Automated disk hygiene cleaning stale crash dumps and logs.
- `agent-relay ctl config-check`: Offline validation of `relay.yaml` detecting unrecognized keys.
- `agent-relay completion [bash|zsh]`: Standalone shell Tab-completion generator.
