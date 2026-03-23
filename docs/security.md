# Security Model

**Spec files have the same trust level as shell scripts.** Before loading any spec, you should understand what it will do:

- `service.command` for native services is split on whitespace and executed directly via `exec.Command`. Shell features such as pipes, redirects, and globbing are not available.
- `env` and injected secret values are passed directly to the process environment.
- `volumes` for container services are mounted as specified, including any host path.
- `args` are passed as additional arguments to the container runtime.

**Only load specs you trust.** Do not load specs from untrusted sources without reviewing them first. The spec directory (`~/.aurelia/services/`) should have permissions that prevent other users from writing to it.

## Authentication

### Unix socket

Access to `~/.aurelia/aurelia.sock` is controlled by filesystem permissions (0600). Only processes running as the same user can connect. No token required.

### TCP API (bearer token)

Required when the daemon is started with `--api-addr` and no TLS config. A 256-bit random token is generated on startup and written to `~/.aurelia/api.token` (0600). All TCP requests must include `Authorization: Bearer <token>`. Constant-time comparison prevents timing attacks.

### TCP API (TLS + mTLS)

When TLS is configured in `~/.aurelia/config.yaml`, the TCP listener uses TLS 1.3 with:

- **Server certificate**: presented to all clients
- **Client certificate verification**: `VerifyClientCertIfGiven`
- **CA certificate**: Vault intermediate CA for verifying client certs

Two authentication paths:

1. **mTLS (daemon-to-daemon)**: peer presents a client cert signed by the same CA. The cert CN identifies the peer. No bearer token needed.
2. **Bearer token (CLI-to-daemon)**: client connects over TLS without a client cert and authenticates via bearer token. Identity recorded as "cli".

```yaml
# ~/.aurelia/config.yaml
tls:
  cert: /path/to/server.crt
  key: /path/to/server.key
  ca: /path/to/ca.crt
```

Certificates are issued by Vault PKI (same infrastructure as managed services). Cert lifecycle is managed outside aurelia.

## Token Rotation

`aurelia token rotate` generates a new token and distributes it to peers:

1. New token generated; old token remains valid (dual-token window)
2. New token pushed to all reachable peers over mTLS (`POST /v1/peer/token`)
3. Each peer updates its local config
4. Once all reachable peers confirm (quorum), old token is invalidated
5. `--force` skips quorum and invalidates immediately

The `POST /v1/peer/token` endpoint requires mTLS authentication and rejects bearer-token clients with 403.

## Rate Limiting

Per-source token bucket rate limiter on the TCP API:

- 20 requests/second sustained, 100 burst
- Keyed by peer identity (cert CN) or remote IP (CLI)
- Returns `429 Too Many Requests` with `Retry-After` header
- Stale limiters cleaned up after 5 minutes

Applied before authentication to protect against brute-force token guessing.

## Audit Logging

All TCP API requests are logged via slog with:

- `peer`: cert CN for mTLS clients, "cli" for bearer token
- `remote_addr`, `method`, `path`, `status`, `duration_ms`

Logged to the daemon's standard log stream (captured by the process supervisor or systemd).

## Cluster Aggregation

The `/v1/cluster/services` endpoint fans out to peers with a 10-second deadline. Peers that don't respond are skipped and reported as "timeout" in the response metadata.

## Trust Boundaries

**Trusted inputs**: service specs (`~/.aurelia/services/*.yaml`) and daemon config (`~/.aurelia/config.yaml`) are loaded from user-writable directories and executed with the user's privileges.

**Untrusted inputs**: the TCP API accepts requests from the network. Binding to a non-loopback address exposes the API. Use TLS when binding to `0.0.0.0`.

**Peer trust**: peers are configured explicitly in `config.yaml`. Only nodes with certs signed by the configured CA are accepted as peers. No automatic peer discovery.

## Network Exposure

- Unix socket (`~/.aurelia/aurelia.sock`): filesystem permissions (0600), local only
- TCP without TLS: bearer token over plaintext. Bind to `127.0.0.1` only. A warning is logged for non-loopback bindings, and a separate warning is logged when TLS is not configured.
- TCP with TLS: encrypted transport, mTLS for peers, bearer token for CLI

## Runtime Input Validation

Service names in API requests are used as map keys, never interpolated into shell commands. Port numbers are validated by `net.Listen`. The lamina remote execution endpoint allowlists subcommands and uses `exec.CommandContext` (no shell interpolation).

**macOS Keychain** stores secrets in the user's login keychain. Secret access is recorded in an append-only audit log at `~/.aurelia/audit.log`.
