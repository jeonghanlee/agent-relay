# Work Register

Release line: master
Milestone index: 2498430
Canonical path: `docs/milestone-2498430.md`
Canonical branch or ref: master
Git upstream: origin/master
Remote tracker: jeonghanlee/agent-relay

Next session entry point: M4 Implementation Plan in `docs/milestone-2498430.md`

## Milestone

### Work

| Group | ID | Work unit | Type | Status | Ready | Deps | Done when / Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Core | M1 | Repository Scaffold, LDFLAGS Build & Docs Trio | Milestone | Complete | No | | `make build` emits version/Git SHA and docs trio complete; [detail](#m1---repository-scaffold-ldflags-build--docs-trio) |
| Core | M2 | Layer 0: Wire Protocol, Framing & Anti-DoS | Milestone | Complete | No | M1 | 4B length-prefix framing & 16MB DoS rejection unit tests pass; [detail](#m2---layer-0-wire-protocol-framing--anti-dos) |
| Core | M3 | Layer 1: Kernel Security, SO_PEERCRED & UDS Socket | Milestone | Complete | No | M2 | SO_PEERCRED UID check & Flock socket recovery tests pass; [detail](#m3---layer-1-kernel-security-so_peercred--uds-socket) |
| Core | M4 | Layer 2: Runtime Supervisor, RLIMIT & Engine | Milestone | Ready | Yes | M3 | Child subreaper orphan tests & offline mock driver pass; [detail](#m4---layer-2-runtime-supervisor-rlimit--engine) |
| Core | M5 | Layer 3: CLI Suite, Ergonomics & Operator Bundle | Milestone | Not started | No | M4 | `ctl abort` < 500ms, man page, shell completion verified; [detail](#m5---layer-3-cli-suite-ergonomics--operator-bundle) |
| Core | M6 | Documentation Book (mdBook) & v1.0.0 Release | Milestone | Not started | No | M5 | `mdbook build` & `tests/verify-book-commands.bash` pass; [detail](#m6---documentation-book-mdbook--v100-release) |

### Decisions

| ID | Decision | Decision Date |
| --- | --- | --- |
| D1 | Use pure Go standard library with vendored golang.org/x/sys/unix for offline air-gapped builds | 2026-09-27 |
| D2 | Separate repository documents into markdown-authoring trio (M1) and mdBook technical book (M6) | 2026-09-27 |
| D3 | Implement evolutionary skeleton main.go in M1 and wire CLI subcommands in M5 | 2026-09-27 |

### Milestone Details

#### M1 - Repository Scaffold, LDFLAGS Build & Docs Trio

Origin: 2498430 / M1
Identity History: none
GitHub Issue: none
Status: Complete

##### Summary

Establish the initial build and documentation foundation for `agent-relay`. Provide a reproducible GNU Makefile, Go 1.22+ module definition with vendored unix syscall package for air-gapped environments, a standalone skeleton `cmd/agent-relay/main.go` supporting build-time LDFLAGS metadata injection, and the canonical engineering documentation trio (`README.md`, `STATUS.md`, `TODO.md`, `docs/ARCHITECTURE.md`) adhering strictly to `markdown-authoring` standards.

##### Scope

- `go.mod` declaration for `agent-relay` and `vendor/golang.org/x/sys/unix` integration
- Modular GNU `Makefile` supporting `build`, `test`, `test-mock`, `clean`, `vendor`, and install targets
- Standalone skeleton `cmd/agent-relay/main.go` handling `--version` and `--help`
- `README.md` with explicit Scope and Out-of-scope declarations
- `STATUS.md` with M1-M6 progress matrix using `✓ ? ✗ —` symbols
- `TODO.md` tracking deferred post-v1.0 items
- `docs/ARCHITECTURE.md` capturing all 22 3rd-person review invariants and 2nd-person operational rules

Out of scope: wire protocol framing implementation, UDS listener sockets, process supervision.

##### Completion Criteria

- `make build` compiles `./bin/agent-relay` with Git commit, build date, and version injected via LDFLAGS.
- `./bin/agent-relay --version` prints valid version metadata and exits 0.
- All documentation trio files exist and pass formatting and ASCII character checks.
- `STATUS.md` reflects M1 as Verified (`✓`) and M2-M6 as Not started (`—`).

##### Dependencies And Decisions

- Resolves D1, D2, D3.

##### Implementation Plan

Plan Status: complete
Plan Acceptance: 2026-09-27 by repository owner
Implementation Authorization: authorized
Superseded Plan Artifacts: none

1. Initialize `go.mod` and vendor `golang.org/x/sys/unix`.
2. Author GNU `Makefile` with standard build flags (`-trimpath -ldflags "-s -w"`).
3. Author `cmd/agent-relay/main.go` baseline skeleton.
4. Author `README.md`, `STATUS.md`, `TODO.md`, and `docs/ARCHITECTURE.md`.
5. Execute verification commands and assert exit codes.

##### Test Plan

| Label | Layer | Method | Environment | Expected Result |
| --- | --- | --- | --- | --- |
| T1 | Build | `make build` | Linux x86_64 | Clean build, binary generated in `bin/agent-relay` |
| T2 | Metadata | `./bin/agent-relay --version` | Linux x86_64 | Version, Git commit, build date printed to stdout |
| T3 | Docs | `grep -rnP '[^\x00-\x7F]' README.md docs/ARCHITECTURE.md` | Linux x86_64 | Zero unauthorized non-ASCII typographic characters |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| T1 | 2026-09-27 16:15 PDT | Linux x86_64 | Passed | `make build` produced `bin/agent-relay` |
| T2 | 2026-09-27 16:15 PDT | Linux x86_64 | Passed | `agent-relay 2498430 (commit: 2498430, built: 2026-09-27T23:14:18Z)` |
| T3 | 2026-09-27 16:16 PDT | Linux x86_64 | Passed | `grep -rnP` returned 0 matches across markdown and Go files |

##### Closure Evidence

- `make build`, `./bin/agent-relay --version`, and `STATUS.md` verification passed on 2026-09-27. Committed as `c8935e3`.

---

#### M2 - Layer 0: Wire Protocol, Framing & Anti-DoS

Origin: 2498430 / M2
Identity History: none
GitHub Issue: none
Status: Complete

##### Summary

Implement Layer 0 core wire protocol, 4-byte big-endian uint32 length-prefixed framing, JSON envelope serialization/deserialization, structured error taxonomy, and defensive anti-DoS barriers (10s read deadline, 16MB allocation limit, 64-reference cardinality limit). Ensure lossless forward compatibility for custom extensions.

##### Scope

- `internal/protocol/`: envelope, message types (`REQUEST`, `EVENT_*`, `RESPONSE`), error codes (`ERR_*`)
- `internal/framing/`: 4-byte length prefix read/write with `io.LimitReader` (max 16MB)
- `test/conformance/`: language-agnostic wire framing test fixtures and roundtrip assertions

Out of scope: UDS socket listener, kernel credentials, process spawning.

##### Completion Criteria

- Unit tests verify 4-byte prefix encoding and decoding for frames under 16MB.
- Frames exceeding 16MB are rejected immediately before heap allocation.
- Lossless forward compatibility preserves unknown fields via `Extensions map[string]json.RawMessage`.
- `go test -race ./internal/protocol/... ./internal/framing/... ./test/conformance/...` passes 100%.

##### Dependencies And Decisions

- Depends on M1.

##### Test Plan

| Label | Layer | Method | Environment | Expected Result |
| --- | --- | --- | --- | --- |
| T1 | Framing | `go test -race ./internal/framing/...` | Linux x86_64 | All framing unit tests pass |
| T2 | Protocol | `go test -race ./internal/protocol/...` | Linux x86_64 | All protocol and error taxonomy tests pass |
| T3 | Conformance | `go test -race ./test/conformance/...` | Linux x86_64 | Wire conformance test fixtures pass |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| T1 | 2026-09-27 16:30 PDT | Linux x86_64 | Passed | `TestFramingRoundtrip_Success`, `TestReadFrame_RejectsExceedingMax_DoSProtection`, deadline tests pass |
| T2 | 2026-09-27 16:30 PDT | Linux x86_64 | Passed | `TestEnvelopeValidation_Success`, missing fields, bounds, and lossless extensions pass |
| T3 | 2026-09-27 16:30 PDT | Linux x86_64 | Passed | `TestWireConformance_AllFrameTypes`, extension roundtrip pass |

##### Closure Evidence

- `go test -race -v ./internal/... ./test/...` executed with 100% pass on 2026-09-27. `make fmt` and `make lint` clean.

---

#### M3 - Layer 1: Kernel Security, SO_PEERCRED & UDS Socket

Origin: 2498430 / M3
Identity History: none
GitHub Issue: none
Status: Complete

##### Summary

Implement Layer 1 host system security and UDS socket listener infrastructure. Enforce kernel UID matching on `Accept()` via `SO_PEERCRED`, sandbox workspace path traversals using `filepath.EvalSymlinks`, prevent symlink race conditions via pre-execution SHA-256 validation, strip unsafe environment variables, and manage single-instance daemon locking with `unix.Flock` and stale socket auto-recovery.

##### Scope

- `internal/security/`: `EvalSymlinks` path jail, caller UID verification, environment allowlist/blacklist
- `internal/socket/`: UDS listener, path fallback (`$XDG_RUNTIME_DIR` -> `/tmp`), `unix.Flock` lockfile, systemd `LISTEN_FDS` socket activation

Out of scope: process execution and log streaming.

##### Completion Criteria

- Connections from differing UIDs are rejected with `ERR_UNAUTHORIZED`.
- File paths resolving outside authorized workspaces are rejected.
- Socket activation detects inherited descriptors when `LISTEN_FDS > 0`.
- Stale lockfiles and dead sockets are safely reclaimed upon daemon startup.
- `go test -race ./internal/security/... ./internal/socket/...` passes 100%.

##### Dependencies And Decisions

- Depends on M2.

##### Test Plan

| Label | Layer | Method | Environment | Expected Result |
| --- | --- | --- | --- | --- |
| T1 | Security | `go test -race ./internal/security/...` | Linux x86_64 | Path traversal and credential checks pass |
| T2 | Socket | `go test -race ./internal/socket/...` | Linux x86_64 | Flock lock, fallback, and listener tests pass |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| T1 | 2026-09-27 | Linux x86_64 | Passed | `go test -race ./internal/security/...` (5 PASS) |
| T2 | 2026-09-27 | Linux x86_64 | Passed | `go test -race ./internal/socket/...` (4 PASS) |

##### Closure Evidence

- `go test -race -v ./internal/security/... ./internal/socket/...` executed with 100% pass on 2026-09-27. `make fmt` and `make lint` clean.

---

#### M4 - Layer 2: Runtime Supervisor, RLIMIT & Engine

Origin: 2498430 / M4
Identity History: none
GitHub Issue: none
Status: Not started

##### Summary

Implement Layer 2 process supervision and single-turn synchronous streaming relay engine. Enforce low-level Linux safety invariants: `PR_SET_CHILD_SUBREAPER` to prevent zombie accumulation, `PR_SET_PDEATHSIG = SIGTERM` to eliminate orphan processes, `PR_SET_DUMPABLE = 0` to safeguard core dumps, hardware resource bounds via `unix.Setrlimit` (4GB RAM, 100MB files, 1024 FDs), and kernel OOM demotion (`oom_score_adj = +500`). Provide `pkg/mock/` for standalone offline verification.

##### Scope

- `internal/supervisor/`: child process spawning, pdeathsig, subreaper, rlimit enforcement, private TMPDIR
- `internal/relay/`: request routing, session addressing (`<session_id>/<agent>`), cancellation propagation, log chunk coalescing (max 64KB, 50ms)
- `pkg/mock/`: zero-dependency mock agent runner for automated integration testing

Out of scope: CLI flag parsing and user-facing terminal presentation.

##### Completion Criteria

- Killing the relay daemon terminates all child agent processes immediately.
- Attempted memory allocation beyond 4GB triggers immediate rlimit termination.
- Zero open file descriptors leak across child processes (`FD_CLOEXEC` verified via `/proc/<pid>/fd/`).
- `make test-mock` executes full end-to-end streaming against `pkg/mock/` and completes successfully in < 3s.

##### Dependencies And Decisions

- Depends on M3.

##### Test Plan

| Label | Layer | Method | Environment | Expected Result |
| --- | --- | --- | --- | --- |
| T1 | Supervisor | `go test -race ./internal/supervisor/...` | Linux x86_64 | Process isolation and rlimit tests pass |
| T2 | Relay | `go test -race ./internal/relay/...` | Linux x86_64 | Synchronous streaming and chunking tests pass |
| T3 | Mock Integration | `make test-mock` | Linux x86_64 | Full offline E2E relay verification passes |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| T1 | Not run | Linux x86_64 | Pending | `go test` |
| T2 | Not run | Linux x86_64 | Pending | `go test` |
| T3 | Not run | Linux x86_64 | Pending | `make test-mock` |

##### Closure Evidence

- Pending M4 completion.

---

#### M5 - Layer 3: CLI Suite, Ergonomics & Operator Bundle

Origin: 2498430 / M5
Identity History: none
GitHub Issue: none
Status: Not started

##### Summary

Assemble the user-facing and operator-facing CLI suite under `cmd/agent-relay/`. Provide `agent-relay daemon` and `agent-relay ctl` subcommands (`send`, `status --json`, `doctor`, `abort`, `prune`, `config-check`, `verify-workspace`). Bundle Section 1 Unix man page (`docs/man/agent-relay.1`), pure Go shell completion generator, commented configuration template `relay.yaml.example`, and pre-validated systemd socket activation unit files.

##### Scope

- `cmd/agent-relay/`: routing, daemon execution, ctl command implementations
- Shell completion: `agent-relay completion bash` and `zsh`
- Systemd bundle: `systemd/agent-relay.socket` and `systemd/agent-relay.service`
- Operations bundle: `relay.yaml.example` and `docs/man/agent-relay.1`

Out of scope: external mdBook documentation book.

##### Completion Criteria

- `agent-relay ctl abort --all` terminates rogue process groups in < 500ms.
- `agent-relay ctl doctor` inspects socket, permissions, and templates with actionable output.
- `agent-relay completion bash` outputs valid, sourceable completion functions.
- `man docs/man/agent-relay.1` renders cleanly via `mandoc` or `man`.
- `systemd-analyze verify systemd/*` validates unit file syntax without warnings.

##### Dependencies And Decisions

- Depends on M4.

##### Test Plan

| Label | Layer | Method | Environment | Expected Result |
| --- | --- | --- | --- | --- |
| T1 | CLI E2E | `go test -race ./cmd/agent-relay/...` | Linux x86_64 | All CLI subcommand integration tests pass |
| T2 | Abort Scram | `agent-relay ctl abort --all` probe | Linux x86_64 | Immediate teardown in < 500ms |
| T3 | Systemd | `systemd-analyze verify systemd/*` | Linux x86_64 | Unit files validate cleanly |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| T1 | Not run | Linux x86_64 | Pending | `go test` |
| T2 | Not run | Linux x86_64 | Pending | Abort benchmark |
| T3 | Not run | Linux x86_64 | Pending | `systemd-analyze verify` |

##### Closure Evidence

- Pending M5 completion.

---

#### M6 - Documentation Book (mdBook) & v1.0.0 Release

Origin: 2498430 / M6
Identity History: none
GitHub Issue: none
Status: Not started

##### Summary

Author the complete `technical-writing` standard documentation book in `book/` using `mdBook`. Implement Diataxis 4-quadrant chapters: Tutorial (5-minute quickstart), Concepts (22 architecture invariants), Procedures (build, systemd, plugins, troubleshooting), and Reference (CLI, config schema, error codes). Implement automated command verification via `tests/verify-book-commands.bash`, configure GitHub Pages workflow per `git-workflow` rules, and execute final v1.0.0 release tagging.

##### Scope

- `book/book.toml` and `book/src/SUMMARY.md`
- Diataxis chapters across tutorial, concepts, procedures, and reference
- `tests/verify-book-commands.bash` automated command verification script
- `.github/workflows/deploy-pages.yml` deployment action with `build_type: workflow` check
- Final v1.0.0 release verification and Git tagging

Out of scope: post-v1.0 multi-host clustering.

##### Completion Criteria

- `mdbook build book` compiles cleanly without broken links or syntax warnings.
- `bash tests/verify-book-commands.bash` runs all documented commands against the built binary and passes 100%.
- `git status` reports a clean working tree matching origin.
- `v1.0.0` signed tag created and pushed.

##### Dependencies And Decisions

- Depends on M5.

##### Integrated Verification

| Source Check | Re-run Trigger | Shared Surface | Release Verification Label | Expected Result | Result Evidence |
| --- | --- | --- | --- | --- | --- |
| M1 / T1 | Tree rebuild | Build system | Release Verification 1 | Binary builds with LDFLAGS metadata | Passed (`make build`) |
| M4 / T3 | Mock E2E | Relay engine | Release Verification 2 | `make test-mock` completes in < 3s | Pending |
| M5 / T1 | CLI E2E | User interface | Release Verification 3 | All `ctl` subcommands function properly | Pending |
| M6 / T1 | Book Build | Documentation | Release Verification 4 | `mdbook build book` exits 0 cleanly | Pending |

##### Verification Results

| Label | Observed At | Environment | Result | Evidence |
| --- | --- | --- | --- | --- |
| Release Verification 1 | 2026-09-27 16:15 PDT | Linux x86_64 | Passed | `make build` |
| Release Verification 2 | Not run | Linux x86_64 | Pending | `make test-mock` |
| Release Verification 3 | Not run | Linux x86_64 | Pending | CLI tests |
| Release Verification 4 | Not run | Linux x86_64 | Pending | `mdbook build book` |

##### Closure Evidence

- Pending M6 completion.

---

## Backlog

### Work

| Group | ID | Work unit | Type | Status | Ready | Deps | Done when / Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Post-v1.0 | B1 | Multi-host TLS overlay bridge | Carry-forward | Deferred | No | D1 | Design document approved; [detail](#b1---multi-host-tls-overlay-bridge) |
| Post-v1.0 | B2 | eBPF socket tracing probes | Carry-forward | Deferred | No | | Kernel eBPF probe suite verified; [detail](#b2---ebpf-socket-tracing-probes) |

### Backlog Details

#### B1 - Multi-host TLS Overlay Bridge

Origin: 2498430 / B1
Status: Deferred (Decision Date: 2026-09-27, deferred to post-v1.0)

##### Summary

Optional TLS/mTLS overlay bridge enabling cross-machine relay clustering across accelerator facility network subnets.

#### B2 - eBPF Socket Tracing Probes

Origin: 2498430 / B2
Status: Deferred (Decision Date: 2026-09-27, deferred to post-v1.0)

##### Summary

Low-overhead eBPF socket filter probes for latency profiling and kernel-level throughput auditing.
