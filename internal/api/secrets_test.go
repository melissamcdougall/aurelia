package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/benaskins/aurelia/internal/keychain"
)

func TestSecretGetFromCache(t *testing.T) {
	srv, client := setupTestServer(t, nil)

	inner := keychain.NewMemoryStore()
	inner.Set("api-key", "secret-value")
	cache := keychain.NewCachedStore(inner, 5*time.Minute)
	srv.SetSecretCache(cache)

	resp, err := client.Get("http://aurelia/v1/secrets/api-key")
	if err != nil {
		t.Fatalf("GET /v1/secrets/api-key: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Value != "secret-value" {
		t.Errorf("expected secret-value, got %q", body.Value)
	}
}

func TestSecretGetNotFound(t *testing.T) {
	srv, client := setupTestServer(t, nil)

	inner := keychain.NewMemoryStore()
	cache := keychain.NewCachedStore(inner, 5*time.Minute)
	srv.SetSecretCache(cache)

	resp, err := client.Get("http://aurelia/v1/secrets/missing")
	if err != nil {
		t.Fatalf("GET /v1/secrets/missing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSecretGetNoCacheConfigured(t *testing.T) {
	_, client := setupTestServer(t, nil)

	resp, err := client.Get("http://aurelia/v1/secrets/any")
	if err != nil {
		t.Fatalf("GET /v1/secrets/any: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}
