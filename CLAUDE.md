@AGENTS.md

## Conventions
- `just build` to build, `just install` to build + install to ~/.local/bin + restart daemon
- Service specs are YAML in ~/.aurelia/services/
- Functional options pattern for Daemon configuration (WithSecrets, WithStateDir, etc.)
- darwin build tags for GPU and Keychain packages; MemoryStore for tests

## Constraints
- darwin/arm64 only — do not add cross-platform abstractions without explicit approval
- Never add axon as a dependency — aurelia supervises axon services but must not import axon's HTTP toolkit
- Each parallel agent MUST use its own git worktree to avoid conflicts
- Commit directly to main — no PRs unless from forks

## Testing
- `just test` for unit tests (short), `just test-all` for all including slow
- `just test-integration` for Docker/OrbStack tests (requires `//go:build integration`)
- `just lint` for go vet
