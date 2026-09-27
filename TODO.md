# Deferred Work and Feature Backlog

## Scope

This document tracks deferred feature additions, extensions, and non-bug increments scheduled for post-v1.0 consideration.

**Out of scope:** Core v1.0 milestones (tracked in `STATUS.md` and `docs/milestone-2498430.md`).

## Deferred Items

### B1. Multi-Host TLS Overlay Bridge

- **Proposal Date:** 2026-09-27
- **Target Line:** Post-v1.0
- **Summary:** Optional mutual TLS (mTLS) TCP overlay bridge allowing multiple `agent-relay` daemons to forward events across physical Linux hosts on accelerator facility networks.
- **Prerequisites:** Complete v1.0 local UDS daemon stabilization.

### B2. eBPF Socket Tracing Probes

- **Proposal Date:** 2026-09-27
- **Target Line:** Post-v1.0
- **Summary:** Optional low-overhead eBPF tracepoints attached to `control.sock` for microsecond-level latency tracking and throughput benchmarking without userspace log parsing.
- **Prerequisites:** Linux 5.15+ kernel baseline verification.
