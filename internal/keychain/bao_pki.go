package keychain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IssuedCert holds the PEM-encoded certificate, private key, and CA chain
// returned by the PKI secrets engine.
type IssuedCert struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	CAChain     string `json:"ca_chain"`
	Serial      string `json:"serial_number"`
	Expiration  int64  `json:"expiration"`
}

// BaoPKIIssuer issues certificates from an OpenBao PKI secrets engine.
type BaoPKIIssuer struct {
	store *BaoStore
	mount string // PKI mount path, e.g. "pki_lamina"
}

// NewBaoPKIIssuer creates an issuer backed by the given BaoStore.
func NewBaoPKIIssuer(store *BaoStore, mount string) *BaoPKIIssuer {
	return &BaoPKIIssuer{store: store, mount: mount}
}

// IssueNodeCert issues a node certificate for mTLS daemon-to-daemon communication.
func (p *BaoPKIIssuer) IssueNodeCert(commonName, ttl string) (*IssuedCert, error) {
	body := fmt.Sprintf(`{"common_name":%q,"ttl":%q}`, commonName, ttl)

	resp, err := p.store.do("PUT", fmt.Sprintf("/v1/%s/issue/node", p.mount), strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pki issue node cert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("pki issue node cert: status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Certificate  string   `json:"certificate"`
			PrivateKey   string   `json:"private_key"`
			CAChain      []string `json:"ca_chain"`
			SerialNumber string   `json:"serial_number"`
			Expiration   int64    `json:"expiration"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("pki issue node cert: decode: %w", err)
	}

	return &IssuedCert{
		Certificate: result.Data.Certificate,
		PrivateKey:  result.Data.PrivateKey,
		CAChain:     strings.Join(result.Data.CAChain, "\n"),
		Serial:      result.Data.SerialNumber,
		Expiration:  result.Data.Expiration,
	}, nil
}
