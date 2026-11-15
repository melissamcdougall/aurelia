---
module: github.com/benaskins/aurelia
kind: supervisor
---

# AGENTS.md

Aurelia is a process supervisor for macOS (a developer-focused alternative to supervisord/launchd). YAML service specs, a daemon, and a CLI that talks to it over a Unix socket (`~/.aurelia/aurelia.sock`). Layers, packages, interfaces, restart policies, signal semantics and runtime files: [docs/internals.md](docs/internals.md). Other docs: [architecture](docs/architecture.md), [api](docs/api.md), [cli](docs/cli-reference.md), [service spec](docs/service-spec.md), [security](docs/security.md).

## Build & Test

First-time setup: `script/bootstrap` (Go, just via Homebrew).

```bash
just build            # Build binary locally
just install          # Build, install to ~/.local/bin, restart daemon
just test             # Unit tests (short)
just test-all         # All tests including slow ones
just test-integration # Docker/OrbStack tests (//go:build integration)
just lint             # gofmt -l check + go vet
just fmt              # go fmt
```

Service specs are YAML in `~/.aurelia/services/`. Tests use the standard `testing` package; helpers use `t.Helper()`/`t.TempDir()`; API tests use `setupTestServer(t, specs)`; `MemoryStore` stands in for Keychain.

## Constraints

- darwin/arm64 only; no cross-platform abstractions without explicit approval.
- cgo for the GPU package; darwin build tags on GPU and Keychain packages.
- Never add axon as a dependency (aurelia supervises axon services but must not import its HTTP toolkit).
- `Daemon` is configured with functional options (`WithSecrets`, `WithStateDir`, `WithPortRange`, `WithRouting`, `WithPeers`).
- Secrets: OpenBao when configured and reachable, else macOS Keychain (`cmd/aurelia/secrets_backend.go`).

## Workflow

Work on a branch in a worktree; parallel agents each get their own. Run `just test-all && just lint` before pushing. Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `infra:`, `config:`. Upstream (`origin`) is `benaskins/aurelia`; our remote is `fork`.

## Agent skills

### Issue tracker

GitHub issues on `melissamcdougall/aurelia`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default triage labels plus `tier:*`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context. See `docs/agents/domain.md`.

## Skills

`aurelia skills` lists embedded skills (`ground-aurelia`, `debug-aurelia`, `deploy-aurelia`); `just install-skills` symlinks them to `.claude/skills/`. Generic skills come from [humanpowers](https://github.com/benaskins/humanpowers).
