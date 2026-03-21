package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/benaskins/aurelia/internal/node"
	"github.com/spf13/cobra"
)

var laminaCmd = &cobra.Command{
	Use:   "lamina <subcommand> [args...]",
	Short: "Run a lamina CLI command (locally or on a remote node)",
	Long: `Execute a lamina workspace command through the aurelia daemon.

Only lamina subcommands are allowed — the daemon validates against
a fixed allowlist. Use --node to target a remote peer.

Examples:
  aurelia lamina repo status
  aurelia lamina --node aurelia repo fetch
  aurelia lamina --node aurelia doctor
  aurelia lamina --node aurelia release axon-chat v0.5.4`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLamina,
}

func init() {
	rootCmd.AddCommand(laminaCmd)
}

func runLamina(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	remote, err := resolveNodeClient(cmd)
	if err != nil {
		return err
	}

	if remote == nil {
		// Local execution — call the local daemon API
		resp, err := postLaminaLocal(args)
		if err != nil {
			return err
		}
		return printLaminaResponse(resp, jsonOut)
	}

	// Remote execution via peer node
	resp, err := remote.Lamina(args)
	if err != nil {
		return err
	}
	return printLaminaResponse(resp, jsonOut)
}

func postLaminaLocal(args []string) (*node.LaminaResponse, error) {
	client, err := apiClient()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]any{"args": args})
	if err != nil {
		return nil, err
	}

	resp, err := client.Post("http://aurelia/v1/lamina", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon: %w (is aurelia daemon running?)", err)
	}
	defer resp.Body.Close()

	var result node.LaminaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

func printLaminaResponse(resp *node.LaminaResponse, jsonOut bool) error {
	if jsonOut {
		return printJSON(resp)
	}

	// Print the command output
	if resp.Output != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(resp.Output)
	} else if resp.Raw != "" {
		fmt.Print(resp.Raw)
	}

	if resp.ExitCode != 0 {
		return fmt.Errorf("lamina exited %d", resp.ExitCode)
	}
	return nil
}
