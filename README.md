# agent-relay

Lightweight Unix-domain socket relay and execution daemon for heterogeneous CLI agents.

## Scope

This repository provides the core daemon (`agent-relay daemon`) and control client (`agent-relay ctl`) responsible for single-turn synchronous streaming message relay, host-level kernel security isolation, and child process lifecycle supervision for autonomous CLI agents.

**Out of scope:** Multi-host network clustering or remote service proxies (addressed separately in post-v1.0 backlog), model evaluation orchestration, and agent prompt engineering.

## Architecture Highlights

- **Wire Framing:** Minimal 4-byte big-endian length-prefixed framing over Unix-domain sockets (`control.sock`).
- **Kernel Security:** Hardware UID matching via `SO_PEERCRED`, symlink path sandboxing via `filepath.EvalSymlinks`, and unsafe environment variable stripping.
- **Process Supervision:** Zombie process prevention with `PR_SET_CHILD_SUBREAPER`, orphan cleanup with `PR_SET_PDEATHSIG = SIGTERM`, core dump protection via `PR_SET_DUMPABLE = 0`, and resource quotas enforced via `unix.Setrlimit` (4GB RAM, 100MB files, 1024 FDs).
- **Hermeticity:** Session-isolated private `TMPDIR` (`0700`), file descriptor leak prevention with `FD_CLOEXEC`, and kernel OOM demotion (`oom_score_adj = +500`).
- **Zero External Dependencies:** Built with pure Go standard library and vendored `golang.org/x/sys/unix` for 100% offline air-gapped compilation.

## Development and Milestones

Development progress and canonical work registers are tracked under:

- `docs/milestone-2498430.md`: Authoritative work register and acceptance gates.
- `docs/ARCHITECTURE.md`: Technical specification capturing all 22 system invariants.
- `STATUS.md`: Live implementation status matrix.
- `TODO.md`: Deferred backlog items.

## Building

Prerequisites: Go 1.22+ and GNU Make.

```bash
make build
```

The compiled binary will be placed at `./bin/agent-relay`. To verify the build:

```bash
./bin/agent-relay --version
```
