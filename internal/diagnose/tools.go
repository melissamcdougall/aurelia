package diagnose

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"

	"github.com/benaskins/aurelia/internal/driver"
	tool "github.com/benaskins/axon-tool"
)

// APIClient abstracts HTTP calls to the aurelia API.
type APIClient interface {
	Get(path string) (*http.Response, error)
	Post(path string) (*http.Response, error)
	Delete(path string) (*http.Response, error)
}

// ConfirmFunc is called before executing an action tool.
// It receives the action, service name, and reason. Returns true to proceed.
type ConfirmFunc func(action, service, reason string) bool

// ReadTools returns the read-only diagnostic tools (no confirmation needed).
func ReadTools(client APIClient) map[string]tool.ToolDef {
	return map[string]tool.ToolDef{
		"list_services":            listServicesTool(client),
		"get_service":              getServiceTool(client),
		"inspect_service":          inspectServiceTool(client),
		"get_logs":                 getLogsTool(client),
		"get_gpu":                  getGPUTool(client),
		"cluster_services":         clusterServicesTool(client),
		"test_health_check":        testHealthCheckTool(client),
		"get_health_check_history": getHealthCheckHistoryTool(client),
		"get_service_dependencies": getServiceDependenciesTool(client),
		"get_system_resources":     getSystemResourcesTool(client),
		"check_port":               checkPortTool(),
	}
}

// ActionTools returns tools that mutate service state (require confirmation).
func ActionTools(client APIClient, confirm ConfirmFunc) map[string]tool.ToolDef {
	return map[string]tool.ToolDef{
		"restart_service": actionTool(client, confirm, "restart"),
		"start_service":   actionTool(client, confirm, "start"),
		"stop_service":    actionTool(client, confirm, "stop"),
		"remove_service":  removeTool(client, confirm),
		"reload_specs":    reloadSpecsTool(client, confirm),
		"kill_process":    killProcessTool(client, confirm),
	}
}

// AllTools returns both read and action tools.
func AllTools(client APIClient, confirm ConfirmFunc) map[string]tool.ToolDef {
	tools := ReadTools(client)
	for k, v := range ActionTools(client, confirm) {
		tools[k] = v
	}
	return tools
}

func apiGet(client APIClient, path string) string {
	resp, err := client.Get(path)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	defer resp.Body.Close()
	return readBody(resp)
}

func apiPost(client APIClient, path string) string {
	resp, err := client.Post(path)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	defer resp.Body.Close()
	return readBody(resp)
}

func readBody(resp *http.Response) string {
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
			return tool.ToolResult{Content: apiGet(client, "/v1/services")}
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
			return tool.ToolResult{Content: apiGet(client, "/v1/services/"+name)}
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
			return tool.ToolResult{Content: apiGet(client, "/v1/services/"+name+"/inspect")}
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
			return tool.ToolResult{Content: apiGet(client, fmt.Sprintf("/v1/services/%s/logs?n=%d", name, lines))}
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
			return tool.ToolResult{Content: apiGet(client, "/v1/gpu")}
		},
	}
}

func clusterServicesTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "cluster_services",
		Description: "List all services across all nodes in the cluster, including peer reachability status.",
		Parameters: tool.ParameterSchema{
			Type:       "object",
			Properties: map[string]tool.PropertySchema{},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			return tool.ToolResult{Content: apiGet(client, "/v1/cluster/services")}
		},
	}
}

func testHealthCheckTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "test_health_check",
		Description: "Trigger a health check for a service and return the detailed result including current status and recent check history with timestamps, latency, and errors.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name": {Type: "string", Description: "Name of the service"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			return tool.ToolResult{Content: apiGet(client, "/v1/services/"+name+"/health")}
		},
	}
}

func getHealthCheckHistoryTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_health_check_history",
		Description: "Get recent health check history for a service showing timestamps, pass/fail status, latency, and error messages for the last 50 checks.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name": {Type: "string", Description: "Name of the service"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			return tool.ToolResult{Content: apiGet(client, "/v1/services/"+name+"/health")}
		},
	}
}

func getServiceDependenciesTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_service_dependencies",
		Description: "Get dependency information for a service: what it depends on (after, requires), what depends on it (dependents), and the cascading impact if it goes down (cascade_impact).",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]tool.PropertySchema{
				"name": {Type: "string", Description: "Name of the service"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			return tool.ToolResult{Content: apiGet(client, "/v1/services/"+name+"/deps")}
		},
	}
}

func getSystemResourcesTool(client APIClient) tool.ToolDef {
	return tool.ToolDef{
		Name:        "get_system_resources",
		Description: "Get system resource usage: CPU load averages, memory usage (total, used, percent), and disk usage (total, used, available, percent).",
		Parameters: tool.ParameterSchema{
			Type:       "object",
			Properties: map[string]tool.PropertySchema{},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			return tool.ToolResult{Content: apiGet(client, "/v1/system")}
		},
	}
}

func apiDelete(client APIClient, path string) string {
	resp, err := client.Delete(path)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	defer resp.Body.Close()
	return readBody(resp)
}

func removeTool(client APIClient, confirm ConfirmFunc) tool.ToolDef {
	return tool.ToolDef{
		Name:        "remove_service",
		Description: "Propose removing a service. Stops the service, archives its spec file, and unloads it from the daemon. The operator must approve before the action is executed.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name", "reason"},
			Properties: map[string]tool.PropertySchema{
				"name":   {Type: "string", Description: "Name of the service"},
				"reason": {Type: "string", Description: "Why this service should be removed"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			reason, _ := args["reason"].(string)

			if confirm != nil && !confirm("remove", name, reason) {
				return tool.ToolResult{Content: fmt.Sprintf("Action rejected by operator: remove %s", name)}
			}

			result := apiDelete(client, fmt.Sprintf("/v1/services/%s", name))
			return tool.ToolResult{Content: fmt.Sprintf("Action executed: remove %s. API response: %s", name, result)}
		},
	}
}

func checkPortTool() tool.ToolDef {
	return tool.ToolDef{
		Name:        "check_port",
		Description: "Check what process is listening on a TCP port. Returns the PID and process name, or indicates the port is free. Use this to diagnose 'address already in use' errors.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"port"},
			Properties: map[string]tool.PropertySchema{
				"port": {Type: "number", Description: "TCP port number to check"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			port, _ := args["port"].(float64)
			if port <= 0 {
				return tool.ToolResult{Content: `{"error": "invalid port number"}`}
			}
			pid := driver.FindPIDOnPort(int(port))
			if pid <= 0 {
				return tool.ToolResult{Content: fmt.Sprintf(`{"port": %d, "status": "free", "message": "nothing listening on port %d"}`, int(port), int(port))}
			}
			name, err := driver.ProcessName(pid)
			if err != nil {
				name = "unknown"
			}
			return tool.ToolResult{Content: fmt.Sprintf(`{"port": %d, "status": "in_use", "pid": %d, "process_name": %q}`, int(port), pid, name)}
		},
	}
}

func reloadSpecsTool(client APIClient, confirm ConfirmFunc) tool.ToolDef {
	return tool.ToolDef{
		Name:        "reload_specs",
		Description: "Propose reloading all service specs from disk. Use this after spec files have been updated to pick up changes without restarting the daemon. The operator must approve before the action is executed.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"reason"},
			Properties: map[string]tool.PropertySchema{
				"reason": {Type: "string", Description: "Why specs need to be reloaded"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			reason, _ := args["reason"].(string)
			if confirm != nil && !confirm("reload_specs", "all", reason) {
				return tool.ToolResult{Content: "Action rejected by operator: reload specs"}
			}
			result := apiPost(client, "/v1/reload")
			return tool.ToolResult{Content: fmt.Sprintf("Action executed: reload specs. API response: %s", result)}
		},
	}
}

func killProcessTool(client APIClient, confirm ConfirmFunc) tool.ToolDef {
	return tool.ToolDef{
		Name:        "kill_process",
		Description: "Propose killing an orphaned process by PID. The process must be related to an aurelia-managed service (holding a known service port or matching a known service PID). Use this to clear orphaned processes that prevent services from starting. The operator must approve before the action is executed.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"pid", "reason"},
			Properties: map[string]tool.PropertySchema{
				"pid":    {Type: "number", Description: "Process ID to kill"},
				"reason": {Type: "string", Description: "Why this process should be killed (include evidence from check_port)"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			pidFloat, _ := args["pid"].(float64)
			reason, _ := args["reason"].(string)
			pid := int(pidFloat)
			if pid <= 1 {
				return tool.ToolResult{Content: `{"error": "refusing to kill PID 0 or 1"}`}
			}
			if pid == os.Getpid() {
				return tool.ToolResult{Content: `{"error": "refusing to kill the aurelia daemon process"}`}
			}

			// Verify the PID is related to an aurelia-managed service.
			if relation := relateToService(client, pid); relation == "" {
				return tool.ToolResult{Content: fmt.Sprintf(`{"error": "PID %d is not related to any aurelia-managed service — refusing to kill"}`, pid)}
			}

			if confirm != nil && !confirm("kill_process", fmt.Sprintf("PID %d", pid), reason) {
				return tool.ToolResult{Content: fmt.Sprintf("Action rejected by operator: kill PID %d", pid)}
			}

			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				return tool.ToolResult{Content: fmt.Sprintf(`{"error": "failed to kill PID %d: %s"}`, pid, err)}
			}
			return tool.ToolResult{Content: fmt.Sprintf(`{"status": "killed", "pid": %d, "signal": "SIGTERM"}`, pid)}
		},
	}
}

// relateToService checks if a PID is related to an aurelia-managed service.
// Returns a description of the relationship (e.g. "service:app-chat (env tag)")
// or empty string if unrelated.
//
// Four checks, strongest first:
//  1. AURELIA_SERVICE env var set on the process (survives exec + reparenting)
//  2. The PID directly matches a known service PID
//  3. The PID is holding a port assigned to a known service
//  4. The PID is holding a port in aurelia's dynamic allocation range (20000-32000)
func relateToService(client APIClient, pid int) string {
	// Check 1: AURELIA_SERVICE env tag — definitive proof of aurelia ownership.
	if svcName := driver.AureliaServiceTag(pid); svcName != "" {
		return fmt.Sprintf("service:%s (env tag)", svcName)
	}

	body := apiGet(client, "/v1/services")
	var services []struct {
		Name string `json:"name"`
		PID  int    `json:"pid"`
		Port int    `json:"port"`
	}
	if err := json.Unmarshal([]byte(body), &services); err != nil {
		return "" // can't verify, refuse
	}

	// Check 2: PID directly matches a known service.
	for _, svc := range services {
		if svc.PID == pid {
			return fmt.Sprintf("service:%s (pid match)", svc.Name)
		}
	}

	// Check 3: PID is holding a port assigned to a known service.
	knownPorts := make(map[int]string, len(services))
	for _, svc := range services {
		if svc.Port > 0 {
			knownPorts[svc.Port] = svc.Name
		}
	}
	for port, svcName := range knownPorts {
		if holderPID := driver.FindPIDOnPort(port); holderPID == pid {
			return fmt.Sprintf("service:%s (holding port %d)", svcName, port)
		}
	}

	// Check 4: PID is listening on a port in aurelia's dynamic range.
	// This catches orphans whose port was reallocated to a different number
	// after a daemon restart. The range 20000-32000 is aurelia's default
	// dynamic allocation range — ports in this range are almost certainly
	// aurelia-managed.
	ports := driver.FindPortsForPID(pid)
	for _, p := range ports {
		if p >= 20000 && p <= 32000 {
			return fmt.Sprintf("aurelia dynamic port range (listening on port %d)", p)
		}
	}

	return ""
}

func actionTool(client APIClient, confirm ConfirmFunc, action string) tool.ToolDef {
	return tool.ToolDef{
		Name:        action + "_service",
		Description: fmt.Sprintf("Propose %sing a service. The operator must approve before the action is executed.", action),
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"name", "reason"},
			Properties: map[string]tool.PropertySchema{
				"name":   {Type: "string", Description: "Name of the service"},
				"reason": {Type: "string", Description: "Why this action is needed"},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			name, _ := args["name"].(string)
			reason, _ := args["reason"].(string)

			if confirm != nil && !confirm(action, name, reason) {
				return tool.ToolResult{Content: fmt.Sprintf("Action rejected by operator: %s %s", action, name)}
			}

			result := apiPost(client, fmt.Sprintf("/v1/services/%s/%s", name, action))
			return tool.ToolResult{Content: fmt.Sprintf("Action executed: %s %s. API response: %s", action, name, result)}
		},
	}
}
