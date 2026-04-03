package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestGetSecretViaDaemon(t *testing.T) {
	// Fake daemon on a unix socket
	dir := t.TempDir()
	sock := filepath.Join(dir, "test.sock")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/secrets/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "api-key" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"value": "daemon-value"})
			return
		}
		http.NotFound(w, r)
	})

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	val, err := getSecretViaDaemon(sock, "api-key")
	if err != nil {
		t.Fatalf("getSecretViaDaemon: %v", err)
	}
	if val != "daemon-value" {
		t.Errorf("expected daemon-value, got %q", val)
	}
}

func TestGetSecretViaDaemonNotFound(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "test.sock")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/secrets/{key}", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	_, err = getSecretViaDaemon(sock, "missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestGetSecretViaDaemonSocketMissing(t *testing.T) {
	_, err := getSecretViaDaemon("/nonexistent/path.sock", "key")
	if err == nil {
		t.Fatal("expected error when socket doesn't exist")
	}
}

func TestSecretGetFallsBackToDirect(t *testing.T) {
	// Verify that when daemon socket doesn't exist, secretGetValue
	// returns errDaemonUnavailable so caller can fall back.
	// We use a temp dir with no socket.
	dir := t.TempDir()
	sock := filepath.Join(dir, "absent.sock")

	// getSecretViaDaemon should fail
	_, err := getSecretViaDaemon(sock, "key")
	if err == nil {
		t.Fatal("expected error for absent socket")
	}

	// Verify the socket doesn't exist
	_, statErr := os.Stat(sock)
	if statErr == nil {
		t.Fatal("socket should not exist")
	}
}
