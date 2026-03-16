---
name: ground-aurelia
description: Use when grounding in an aurelia-managed environment. Extends /ground with aurelia CLI and service mesh context.
---

# Ground Aurelia

Follow `/ground` — here's how to orient in the aurelia service mesh.

## Orientation

```bash
aurelia status              # All services — PID, port, health, uptime, restarts
aurelia logs <service>      # Tail service output
aurelia list                # List registered service specs
```

## Service specs

Specs live at `~/.aurelia/services/<name>.yaml`. Each defines the service type, port, health check, routing, and restart policy.

Key things to check when grounding:
- Which services are running vs stopped vs failed
- Which services have routing (`.studio.internal` domains via Traefik)
- Which services use dynamic ports (`port: 0` → reads `$PORT`)

## Key commands

```bash
aurelia status              # What's running
aurelia up <service>        # Start
aurelia down <service>      # Stop
aurelia restart <service>   # Restart (config change)
aurelia deploy <service>    # Zero-downtime redeploy (code change)
aurelia reload              # Re-read all specs from disk
aurelia check <spec>        # Validate a spec file
```

$ARGUMENTS
