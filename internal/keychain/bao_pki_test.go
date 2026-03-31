package keychain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBaoPKIIssuer_IssueNodeCert(t *testing.T) {
	expiration := time.Now().Add(72 * time.Hour).Unix()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/v1/pki_lamina/issue/node" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body struct {
			CommonName string `json:"common_name"`
			TTL        string `json:"ttl"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.CommonName != "hestia" {
			t.Errorf("common_name = %q, want %q", body.CommonName, "hestia")
		}
		if body.TTL != "72h" {
			t.Errorf("ttl = %q, want %q", body.TTL, "72h")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"certificate":   "-----BEGIN CERTIFICATE-----\nfake-cert\n-----END CERTIFICATE-----",
				"private_key":   "-----BEGIN EC PRIVATE KEY-----\nfake-key\n-----END EC PRIVATE KEY-----",
				"ca_chain":      []string{"-----BEGIN CERTIFICATE-----\nintermediate\n-----END CERTIFICATE-----", "-----BEGIN CERTIFICATE-----\nroot\n-----END CERTIFICATE-----"},
				"serial_number": "aa:bb:cc",
				"expiration":    expiration,
			},
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	store := NewBaoStore(srv.URL, "test-token", "secret")
	issuer := NewBaoPKIIssuer(store, "pki_lamina")

	cert, err := issuer.IssueNodeCert("hestia", "72h")
	if err != nil {
		t.Fatalf("IssueNodeCert: %v", err)
	}

	if !strings.Contains(cert.Certificate, "fake-cert") {
		t.Errorf("certificate missing expected content")
	}
	if !strings.Contains(cert.PrivateKey, "fake-key") {
		t.Errorf("private key missing expected content")
	}
	if !strings.Contains(cert.CAChain, "intermediate") || !strings.Contains(cert.CAChain, "root") {
		t.Errorf("CA chain missing expected content: %s", cert.CAChain)
	}
	if cert.Serial != "aa:bb:cc" {
		t.Errorf("serial = %q, want %q", cert.Serial, "aa:bb:cc")
	}
	if cert.Expiration != expiration {
		t.Errorf("expiration = %d, want %d", cert.Expiration, expiration)
	}
}

func TestBaoPKIIssuer_IssueNodeCertError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	store := NewBaoStore(srv.URL, "test-token", "secret")
	issuer := NewBaoPKIIssuer(store, "pki_lamina")

	_, err := issuer.IssueNodeCert("unknown-node", "72h")
	if err == nil {
		t.Error("expected error for forbidden request")
	}
}
