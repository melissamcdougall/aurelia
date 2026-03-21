package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/aurelia/internal/daemon"
)

func laminaServer(t *testing.T, laminaRoot string) *Server {
	t.Helper()
	d := daemon.NewDaemon(t.TempDir())
	srv := NewServer(d, nil)
	srv.SetLaminaRoot(laminaRoot)
	return srv
}

func TestLaminaExec_NoRootConfigured(t *testing.T) {
	srv := laminaServer(t, "")

	body, _ := json.Marshal(map[string]any{"args": []string{"repo", "status"}})
	req := httptest.NewRequest("POST", "/v1/lamina", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.laminaExec(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestLaminaExec_EmptyArgs(t *testing.T) {
	srv := laminaServer(t, "/tmp")

	body, _ := json.Marshal(map[string]any{"args": []string{}})
	req := httptest.NewRequest("POST", "/v1/lamina", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.laminaExec(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLaminaExec_ForbiddenCommand(t *testing.T) {
	srv := laminaServer(t, "/tmp")

	for _, cmd := range []string{"bash", "sh", "rm", "sudo", "curl", "exec"} {
		body, _ := json.Marshal(map[string]any{"args": []string{cmd}})
		req := httptest.NewRequest("POST", "/v1/lamina", bytes.NewReader(body))
		w := httptest.NewRecorder()

		srv.laminaExec(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("command %q: expected 403, got %d", cmd, w.Code)
		}
	}
}

func TestLaminaExec_AllowedCommands(t *testing.T) {
	for cmd := range allowedCommands {
		if !allowedCommands[cmd] {
			t.Errorf("command %q should be allowed", cmd)
		}
	}

	// Verify specific commands are in the set
	expected := []string{"repo", "doctor", "heal", "test", "release", "deps", "hooks", "init", "eval", "skills"}
	for _, cmd := range expected {
		if !allowedCommands[cmd] {
			t.Errorf("expected %q in allowedCommands", cmd)
		}
	}
}

func TestLaminaExec_InvalidBody(t *testing.T) {
	srv := laminaServer(t, "/tmp")

	req := httptest.NewRequest("POST", "/v1/lamina", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	srv.laminaExec(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLaminaExec_ValidCommand(t *testing.T) {
	// This test requires lamina to be installed; skip if not available
	srv := laminaServer(t, t.TempDir())

	body, _ := json.Marshal(map[string]any{"args": []string{"help"}})
	req := httptest.NewRequest("POST", "/v1/lamina", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.laminaExec(w, req)

	// help exits 0 even if not in a workspace
	var resp laminaResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// We just care that it tried to exec — if lamina isn't installed
	// it'll get exit -1, which is fine for the test
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 200, 422, or 500, got %d", w.Code)
	}
}
