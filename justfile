version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

build:
    go build -ldflags "-X main.version={{version}}" -o aurelia ./cmd/aurelia/

test:
    go test -short ./...

test-all:
    go test ./...

test-integration:
    go test -tags integration ./...

lint: fmt-check
    go vet ./...

fmt:
    go fmt ./...

fmt-check:
    @files=$(gofmt -l .); \
    if [ -n "$files" ]; then \
        echo "gofmt drift detected:"; \
        echo "$files"; \
        echo "Run: just fmt"; \
        exit 1; \
    fi

build-lean:
    go build -tags nocontainer,nogpu -ldflags "-s -w -X main.version={{version}}" -o aurelia-lean ./cmd/aurelia/

test-examples:
    docker build -f examples/Dockerfile -t aurelia-examples-test .
    docker run --rm aurelia-examples-test

install: build
    mv aurelia ~/.local/bin/aurelia-dev
    ln -sfn aurelia-dev ~/.local/bin/aurelia
    launchctl stop com.aurelia.daemon
    @echo "Installed aurelia-dev {{version}} and restarted daemon"

use-release:
    go install github.com/benaskins/aurelia/cmd/aurelia@latest
    cp ~/go/bin/aurelia ~/.local/bin/aurelia-release
    ln -sfn aurelia-release ~/.local/bin/aurelia
    launchctl stop com.aurelia.daemon
    @echo "Switched to aurelia-release and restarted daemon"

use-dev:
    #!/usr/bin/env bash
    if [[ ! -f ~/.local/bin/aurelia-dev ]]; then
        echo "Error: No dev build found. Run 'just install' first."
        exit 1
    fi
    ln -sfn aurelia-dev ~/.local/bin/aurelia
    launchctl stop com.aurelia.daemon
    echo "Switched to aurelia-dev and restarted daemon"

which:
    #!/usr/bin/env bash
    echo "Active: $(readlink ~/.local/bin/aurelia 2>/dev/null || echo 'not a symlink')"
    echo "Version: $(~/.local/bin/aurelia --version 2>/dev/null || echo 'unknown')"
    echo ""
    if [[ -f ~/.local/bin/aurelia-dev ]]; then
        echo "Dev:     $(~/.local/bin/aurelia-dev --version 2>/dev/null || echo 'unknown')"
    else
        echo "Dev:     not installed"
    fi
    if [[ -f ~/.local/bin/aurelia-release ]]; then
        echo "Release: $(~/.local/bin/aurelia-release --version 2>/dev/null || echo 'unknown')"
    else
        echo "Release: not installed"
    fi

migrate-daemon:
    #!/usr/bin/env bash
    set -euo pipefail

    PLIST=~/Library/LaunchAgents/com.aurelia.daemon.plist

    # Check if already migrated
    if grep -q '/.local/bin/aurelia' "$PLIST" 2>/dev/null; then
        echo "Already migrated. LaunchAgent points at ~/.local/bin/aurelia."
        just which
        exit 0
    fi

    echo "=== Aurelia daemon migration ==="

    # 1. Build dev binary from current checkout
    echo "Building dev binary..."
    just build
    mv aurelia ~/.local/bin/aurelia-dev

    # 2. Fetch latest release
    echo "Fetching latest upstream release..."
    go install github.com/benaskins/aurelia/cmd/aurelia@latest
    cp ~/go/bin/aurelia ~/.local/bin/aurelia-release

    # 3. Default to release (safe)
    echo "Creating symlink (defaulting to release)..."
    ln -sfn aurelia-release ~/.local/bin/aurelia

    # 4. Update LaunchAgent
    echo "Updating LaunchAgent plist..."
    launchctl bootout gui/$(id -u)/com.aurelia.daemon 2>/dev/null || true
    sed -i '' 's|/Users/melissamcdougall/go/bin/aurelia|/Users/melissamcdougall/.local/bin/aurelia|' "$PLIST"
    launchctl bootstrap gui/$(id -u) "$PLIST"

    echo ""
    echo "Migration complete. Daemon restarted with release binary."
    just which

clean:
    rm -f aurelia aurelia-lean

install-hooks:
    printf '#!/bin/sh\ngofmt -w .\ngit diff --quiet || { echo "gofmt reformatted files — re-stage and commit again"; exit 1; }\n\n# slop guard\nif command -v slop-guard >/dev/null 2>&1; then\n  slop-guard --staged || exit 1\nfi\n' > .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit

# Symlink skills from skills/ into .claude/skills/ for Claude Code discovery
install-skills:
    mkdir -p .claude/skills
    for dir in skills/*/; do \
        name=$(basename "$dir"); \
        ln -sfn "$(pwd)/$dir" ".claude/skills/$name"; \
    done
    @echo "Installed $(ls -1 skills/ | wc -l | tr -d ' ') skill(s) to .claude/skills/"
