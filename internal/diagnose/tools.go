package diagnose

import (
	"fmt"
	"io"
	"net/http"

	tool "github.com/benaskins/axon-tool"
)

// APIClient abstracts HTTP calls to the aurelia API.
type APIClient interface {
	Get(path string) (*http.Response, error)
}

// Tools returns the set of diagnostic tools that query aurelia's API.
func Tools(client APIClient) map[string]tool.ToolDef {
	return map[string]tool.ToolDef{
		"list_services":   listServicesTool(client),
		"get_service":     getServiceTool(client),
		"inspect_service": inspectServiceTool(client),
		"get_logs":        getLogsTool(client),
		"get_gpu":         getGPUTool(client),
	}
}

func apiCall(client APIClient, path string) string {
	resp, err := client.Get(path)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	if resp.StatusCode >= 400 {
		return fmt.Sprintf(`{"error": "API returned status %d", "body": %s}`, resp.StatusCode, body)
	}
	return string(body)
}

func listServicesTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "list_services",
		Description: "List all services managed by aurelia with their state, health, PID, port, uptime, and restart count.",
		Parameters: tool.ParameterSchema{
			Type:       "object",
			Properties: map[string]tool.PropertySchema{},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			return tool.ToolResult{Content: apiCall(client, "/v1/services")}
		},
	}
}

func getServiceTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_service",
		Description: "Get the current state of a specific service including health status, PID, port, uptime, restart count, and last error.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name": {Type: "string", Description: "Name of the service"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			return tool.ToolResult{Content: apiCall(client, "/v1/services/"+name)}
		},
	}
}

func inspectServiceTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "inspect_service",
		Description: "Get full resolved config and runtime state for a service, including command, env vars, health check config, restart policy, dependencies, and routing.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name": {Type: "string", Description: "Name of the service"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			return tool.ToolResult{Content: apiCall(client, "/v1/services/"+name+"/inspect")}
		},
	}
}

func getLogsTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_logs",
		Description: "Get recent log output from a service. Returns the last N lines from the service's stdout/stderr.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name":  {Type: "string", Description: "Name of the service"},
				"lines": {Type: "number", Description: "Number of log lines to retrieve (default 100)", Default: 100},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			lines := 100
			if n, ok := args["lines"].(float64); ok && n > 0 {
				lines = int(n)
			}
			return tool.ToolResult{Content: apiCall(client, fmt.Sprintf("/v1/services/%s/logs?n=%d", name, lines))}
		},
	}
}

func getGPUTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_gpu",
		Description: "Get Apple Silicon GPU metrics including VRAM usage, thermal state, and GPU model.",
		Parameters: tool.ParameterSchema{
			Type:       "object",
			Properties: map[string]tool.PropertySchema{},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			return tool.ToolResult{Content: apiCall(client, "/v1/gpu")}
		},
	}
}

// ToolDefs returns the tool definitions as a slice (for axon-loop requests).
func ToolDefs(client APIClient) []tool.ToolDef {
	tools := Tools(client)
	defs := make([]tool.ToolDef, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, t)
	}
	return defs
}

