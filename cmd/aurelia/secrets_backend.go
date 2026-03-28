package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/benaskins/aurelia/internal/audit"
	"github.com/benaskins/aurelia/internal/config"
	"github.com/benaskins/aurelia/internal/keychain"
)

// newSecretStore creates the secret store using the configured backend.
// It prefers OpenBao when configured and reachable, falling back to macOS Keychain.
func newSecretStore(actor string) (*keychain.AuditedStore, error) {
	dir, err := aureliaHome()
	if err != nil {
		return nil, fmt.Errorf("finding aurelia home: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	auditLog, err := audit.NewLogger(filepath.Join(dir, "audit.log"))
	if err != nil {
		return nil, err
	}

	meta, err := keychain.NewMetadataStore(filepath.Join(dir, "secret-metadata.json"))
	if err != nil {
		return nil, err
	}

	inner, err := resolveBackend(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving secrets backend: %w", err)
	}
	return keychain.NewAuditedStore(inner, auditLog, meta, actor), nil
}

// resolveBackend picks the best available secrets backend.
// When OpenBao is configured, it is required — no silent fallback to Keychain.
func resolveBackend(stateDir string) (keychain.Store, error) {
	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config %s: %w", cfgPath, err)
	}

	if cfg.OpenBao != nil {
		token, err := cfg.OpenBao.LoadToken()
		if err != nil {
			return nil, fmt.Errorf("openbao configured but token not available: %w", err)
		}

		var opts []keychain.BaoOption
		if cfg.OpenBao.UnsealFile != "" {
			opts = append(opts, keychain.WithUnsealFile(cfg.OpenBao.UnsealFile))
		}

		mount := cfg.OpenBao.Mount
		if mount == "" {
			mount = "secret"
		}

		store := keychain.NewBaoStore(cfg.OpenBao.Addr, token, mount, opts...)
		if err := store.Ping(); err != nil {
			return nil, fmt.Errorf("openbao configured but unreachable at %s: %w", cfg.OpenBao.Addr, err)
		}

		slog.Info("secrets backend: openbao", "addr", cfg.OpenBao.Addr)
		return store, nil
	}

	return keychain.NewSystemStore(), nil
}
