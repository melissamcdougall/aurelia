package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/benaskins/aurelia/internal/config"

	"github.com/spf13/cobra"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage TLS certificates",
}

var certRenewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Renew the wildcard TLS certificate",
	Long: `Issue a new *.hestia.internal wildcard certificate from the PKI
secrets engine on the CA node (adyton), write it to the data directory,
and reload Traefik to pick up the new certificate.`,
	RunE: runCertRenew,
}

var certIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue a certificate via the CA node",
	Long: `Request a certificate from the CA node's PKI secrets engine.
Supports server and client certificates with configurable role, CN, and TTL.

Examples:
  aurelia cert issue --role server --cn "*.hestia.internal" --cert-dir data/vault/server-certs/wildcard
  aurelia cert issue --role client --cn chat-client --cert-dir data/vault/client-certs/chat-client`,
	RunE: runCertIssue,
}

var certBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Output the CA certificate chain for browser import",
	Long: `Print the CA chain PEM to stdout. Pipe to a file and import
into your browser's certificate store to trust *.hestia.internal.

  aurelia cert bundle > lamina-ca.crt`,
	RunE: runCertBundle,
}

func init() {
	certRenewCmd.Flags().String("ttl", "720h", "Certificate time to live")
	certRenewCmd.Flags().String("role", "server", "PKI role to issue against")
	certRenewCmd.Flags().String("cn", "*.hestia.internal", "Common name for the certificate")

	certIssueCmd.Flags().String("ttl", "720h", "Certificate time to live")
	certIssueCmd.Flags().String("role", "server", "PKI role (server, client, node)")
	certIssueCmd.Flags().String("cn", "", "Common name for the certificate (required)")
	certIssueCmd.Flags().String("cert-dir", "", "Directory to write cert files (required)")
	certIssueCmd.MarkFlagRequired("cn")
	certIssueCmd.MarkFlagRequired("cert-dir")

	certCmd.AddCommand(certRenewCmd)
	certCmd.AddCommand(certIssueCmd)
	certCmd.AddCommand(certBundleCmd)
	rootCmd.AddCommand(certCmd)
}

// resolveCAClient builds a peer client to the CA node (adyton).
func resolveCAClient() (*config.Config, error) {
	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config %s: %w", cfgPath, err)
	}
	return cfg, nil
}

func issueCertViaPeer(role, cn, ttl string) (*certResult, error) {
	cfg, err := resolveCAClient()
	if err != nil {
		return nil, err
	}

	// Determine the CA peer — use OpenBaoPeer config if available,
	// otherwise fall back to OpenBao (local mode on adyton itself).
	var peerName string
	if cfg.OpenBaoPeer != nil {
		peerName = cfg.OpenBaoPeer.Peer
	} else {
		return nil, fmt.Errorf("no openbao_peer configured — cannot reach CA node")
	}

	peer, err := buildPeerClient(cfg, peerName)
	if err != nil {
		return nil, fmt.Errorf("building peer client to %s: %w", peerName, err)
	}

	resp, err := peer.IssueCert(role, cn, ttl)
	if err != nil {
		return nil, fmt.Errorf("issuing cert via %s: %w", peerName, err)
	}

	return &certResult{
		Certificate: resp.Certificate,
		PrivateKey:  resp.PrivateKey,
		CAChain:     resp.CAChain,
		Serial:      resp.Serial,
		Expiration:  resp.Expiration,
	}, nil
}

type certResult struct {
	Certificate string
	PrivateKey  string
	CAChain     string
	Serial      string
	Expiration  int64
}

func writeCertFiles(dir string, cert *certResult, isClient bool) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cert dir: %w", err)
	}

	if isClient {
		if err := os.WriteFile(filepath.Join(dir, "client.crt"), []byte(cert.Certificate), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "client.key"), []byte(cert.PrivateKey), 0600); err != nil {
			return err
		}
	} else {
		if err := os.WriteFile(filepath.Join(dir, "cert.crt"), []byte(cert.Certificate), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "cert.key"), []byte(cert.PrivateKey), 0600); err != nil {
			return err
		}
		fullchain := cert.Certificate + "\n" + cert.CAChain
		if err := os.WriteFile(filepath.Join(dir, "fullchain.crt"), []byte(fullchain), 0644); err != nil {
			return err
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "ca-chain.crt"), []byte(cert.CAChain), 0644); err != nil {
		return err
	}
	return nil
}

func runCertRenew(cmd *cobra.Command, _ []string) error {
	ttl, _ := cmd.Flags().GetString("ttl")
	role, _ := cmd.Flags().GetString("role")
	cn, _ := cmd.Flags().GetString("cn")
	jsonOut, _ := cmd.Flags().GetBool("json")

	certDir, err := wildcardCertDir()
	if err != nil {
		return err
	}

	fmt.Printf("Issuing %s (role=%s, ttl=%s) via CA node...\n", cn, role, ttl)

	cert, err := issueCertViaPeer(role, cn, ttl)
	if err != nil {
		return err
	}

	if err := writeCertFiles(certDir, cert, false); err != nil {
		return err
	}

	expiry := time.Unix(cert.Expiration, 0)

	if jsonOut {
		return printJSON(map[string]any{
			"common_name": cn,
			"serial":      cert.Serial,
			"expires":     expiry.Format(time.RFC3339),
			"cert_dir":    certDir,
		})
	}

	fmt.Printf("Certificate issued: %s\n", cn)
	fmt.Printf("  Serial:  %s\n", cert.Serial)
	fmt.Printf("  Expires: %s\n", expiry.Format(time.RFC3339))
	fmt.Printf("  Dir:     %s\n", certDir)

	// Reload traefik to pick up the new cert
	fmt.Print("Reloading traefik...")
	if _, err := apiPost("/v1/services/infra-traefik/restart"); err != nil {
		fmt.Printf(" failed: %v\n", err)
		fmt.Println("Restart traefik manually: aurelia restart infra-traefik")
	} else {
		fmt.Println(" done")
	}

	return nil
}

func runCertIssue(cmd *cobra.Command, _ []string) error {
	ttl, _ := cmd.Flags().GetString("ttl")
	role, _ := cmd.Flags().GetString("role")
	cn, _ := cmd.Flags().GetString("cn")
	certDir, _ := cmd.Flags().GetString("cert-dir")
	jsonOut, _ := cmd.Flags().GetBool("json")

	fmt.Printf("Issuing %s (role=%s, ttl=%s) via CA node...\n", cn, role, ttl)

	cert, err := issueCertViaPeer(role, cn, ttl)
	if err != nil {
		return err
	}

	isClient := role == "client"
	if err := writeCertFiles(certDir, cert, isClient); err != nil {
		return err
	}

	expiry := time.Unix(cert.Expiration, 0)

	if jsonOut {
		return printJSON(map[string]any{
			"common_name": cn,
			"serial":      cert.Serial,
			"expires":     expiry.Format(time.RFC3339),
			"cert_dir":    certDir,
		})
	}

	fmt.Printf("Certificate issued: %s\n", cn)
	fmt.Printf("  Serial:  %s\n", cert.Serial)
	fmt.Printf("  Expires: %s\n", expiry.Format(time.RFC3339))
	fmt.Printf("  Dir:     %s\n", certDir)

	return nil
}

func runCertBundle(cmd *cobra.Command, _ []string) error {
	certDir, err := wildcardCertDir()
	if err != nil {
		return err
	}

	caChain, err := os.ReadFile(filepath.Join(certDir, "ca-chain.crt"))
	if err != nil {
		return fmt.Errorf("reading CA chain: %w (run 'aurelia cert renew' first)", err)
	}

	fmt.Print(string(caChain))
	return nil
}

func wildcardCertDir() (string, error) {
	root := os.Getenv("AURELIA_ROOT")
	if root == "" {
		return "", fmt.Errorf("AURELIA_ROOT is not set")
	}
	return filepath.Join(root, "data", "vault", "server-certs", "wildcard"), nil
}
