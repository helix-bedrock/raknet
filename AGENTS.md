# AGENTS.md

Rules for AI coding agents working in this repo. Also read the org-wide rules in
`../.github/AGENT_RULES.md` (online:
<https://github.com/helix-bedrock/.github/blob/main/AGENT_RULES.md>). Where they
disagree, this file wins for this repo.

## What this repo is

The RakNet transport for Helix: UDP connections, reliability, ordering,
splitting, congestion control and metrics for Minecraft Bedrock. Used by the
Helix server; the client dialer is kept but is not the focus.

## Commands

```bash
go build ./...
go vet ./...
golangci-lint run
go test -race ./...
```

## Layout

- Root package: `Listener`, `Conn`, `Dialer`, reliability, congestion, metrics.
- `internal/message`: RakNet offline and connected control messages.

## Invariants: do not break these

1. Wire format matches vanilla Bedrock RakNet byte for byte.
2. The original copyright line in `LICENSE` stays.
3. Hot paths (packet read and write) must not allocate in the steady state.

## Gotchas

- Imported from lunar-bedrock/go-raknet (a go-raknet fork); see `CREDITS.md`. Low-level hooks (connection
  attempts, frames, raw bytes) are being added; see the Helix API design.
