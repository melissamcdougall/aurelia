package diagnose

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	tool "github.com/benaskins/axon-tool"
)

// testAPIClient wraps an httptest.Server to implement APIClient.
type testAPIClient struct {
	server *httptest.Server
}

func (c *testAPIClient) Get(path string) (*http.Response, error) {
	return http.Get(c.server.URL + path)
}

func setupTestAPI(t *testing.T, handler http.Handler) APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &testAPIClient{server: server}
}

func TestToolsReturnsAllTools(t *testing.T) {
	t.Parallel()
	client := setupTestAPI(t, http.NotFoundHandler())
	tools := Tools(client)

	expected := []string{"list_services", "get_service", "inspect_service", "get_logs", "get_gpu"}
	for _, name := range expected {
		if _, ok := tools[name]; !ok {
			t.Errorf("missing tool %q", name)
		}
	}
	if len(tools) != len(expected) {
		t.Errorf("got %d tools, want %d", len(tools), len(expected))
	}
}

func TestListServicesTool(t *testing.T) {
	t.Parallel()
	services := []map[string]any{
		{"name": "chat", "state": "running", "health": "healthy"},
		{"name": "auth", "state": "running", "health": "healthy"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/services", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(services)
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)
	result := tools["list_services"].Execute(&tool.ToolContext{}, nil)

	var got []map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v\ncontent: %s", err, result.Content)
	}
	if len(got) != 2 {
		t.Errorf("got %d services, want 2", len(got))
	}
}

func TestGetServiceTool(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/services/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": name, "state": "running"})
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)
	result := tools["get_service"].Execute(&tool.ToolContext{}, map[string]any{"name": "chat"})

	var got map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if got["name"] != "chat" {
		t.Errorf("name = %v, want chat", got["name"])
	}
}

func TestGetLogsTool(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/services/{name}/logs", func(w http.ResponseWriter, r *http.Request) {
		n := r.URL.Query().Get("n")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"lines": []string{"line1", "line2"}, "requested": n})
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)

	// Default lines
	result := tools["get_logs"].Execute(&tool.ToolContext{}, map[string]any{"name": "chat"})
	var got map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if got["requested"] != "100" {
		t.Errorf("requested = %v, want 100", got["requested"])
	}

	// Custom lines
	result2 := tools["get_logs"].Execute(&tool.ToolContext{}, map[string]any{"name": "chat", "lines": float64(50)})
	var got2 map[string]any
	if err := json.Unmarshal([]byte(result2.Content), &got2); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if got2["requested"] != "50" {
		t.Errorf("requested = %v, want 50", got2["requested"])
	}
}

func TestGetGPUTool(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/gpu", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": "Apple M2 Ultra", "vram_total_bytes": 196608000000})
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)
	result := tools["get_gpu"].Execute(&tool.ToolContext{}, nil)

	var got map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if got["name"] != "Apple M2 Ultra" {
		t.Errorf("name = %v, want Apple M2 Ultra", got["name"])
	}
}

func TestInspectServiceTool(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/services/{name}/inspect", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": name, "command": "/usr/bin/chat", "state": "running"})
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)
	result := tools["inspect_service"].Execute(&tool.ToolContext{}, map[string]any{"name": "chat"})

	var got map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if got["command"] != "/usr/bin/chat" {
		t.Errorf("command = %v, want /usr/bin/chat", got["command"])
	}
}

func TestAPICallError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/services/{name}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "service not found"})
	})

	client := setupTestAPI(t, mux)
	tools := Tools(client)
	result := tools["get_service"].Execute(&tool.ToolContext{}, map[string]any{"name": "nonexistent"})

	var got map[string]any
	if err := json.Unmarshal([]byte(result.Content), &got); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if _, ok := got["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestToolDefsReturnsSlice(t *testing.T) {
	t.Parallel()
	client := setupTestAPI(t, http.NotFoundHandler())
	defs := ToolDefs(client)
	if len(defs) != 5 {
		t.Errorf("got %d tool defs, want 5", len(defs))
	}
}
