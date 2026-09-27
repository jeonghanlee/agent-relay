# Implementation Status

## Scope

This document tracks implementation progress across the six core milestones for `agent-relay`.

**Out of scope:** Historical changelogs and issue discussions (managed in git commit history and GitHub issues).

## Coverage Matrix

| Milestone | Description | Layer | Status | Target Deliverable |
| --- | --- | --- | --- | --- |
| M1 | Repository Scaffold, LDFLAGS Build & Docs Trio | Infra | ✓ | `Makefile`, `go.mod`, docs trio, `bin/agent-relay` |
| M2 | Layer 0: Wire Protocol, Framing & Anti-DoS | Layer 0 | — | `internal/protocol/`, `internal/framing/` |
| M3 | Layer 1: Kernel Security, SO_PEERCRED & Socket | Layer 1 | — | `internal/security/`, `internal/socket/` |
| M4 | Layer 2: Runtime Supervisor, RLIMIT & Engine | Layer 2 | — | `internal/supervisor/`, `internal/relay/`, `pkg/mock/` |
| M5 | Layer 3: CLI Suite, Ergonomics & Operator Bundle | Layer 3 | — | `cmd/agent-relay/`, `docs/man/`, `systemd/` |
| M6 | Documentation Book (mdBook) & v1.0.0 Release | Release | — | `book/`, `tests/verify-book-commands.bash` |

*Status Key:*
- `✓`: Verified and complete (acceptance gate passed).
- `?`: In progress or awaiting verification.
- `✗`: Defect identified / verification failed.
- `—`: Not started.

## Open Items

None. Milestone M1 build and documentation verification passed cleanly.

## Update Protocol

This status matrix and any active Open Items must be updated in the exact same commit that introduces the corresponding substantive code or verification change.
