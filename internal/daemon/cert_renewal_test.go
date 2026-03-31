package daemon

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benaskins/aurelia/internal/keychain"
	"github.com/benaskins/aurelia/internal/node"
)

// testCertKit generates a CA and node cert for testing cert renewal.
type testCertKit struct {
	CACert *x509.Certificate
	CAKey  *ecdsa.PrivateKey
	Dir    string
}

func newTestCertKit(t *testing.T) *testCertKit {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatal(err)
	}

	// Write CA cert
	writePEMToFile(t, filepath.Join(dir, "ca-chain.crt"), "CERTIFICATE", caCertDER)

	return &testCertKit{CACert: caCert, CAKey: caKey, Dir: dir}
}

func (k *testCertKit) issueNodeCert(t *testing.T, name string, notBefore, notAfter time.Time) {
	t.Helper()
	nodeKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{name},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, k.CACert, &nodeKey.PublicKey, k.CAKey)
	if err != nil {
		t.Fatal(err)
	}

	writePEMToFile(t, filepath.Join(k.Dir, "node.crt"), "CERTIFICATE", certDER)

	keyDER, err := x509.MarshalECPrivateKey(nodeKey)
	if err != nil {
		t.Fatal(err)
	}
	writePEMToFile(t, filepath.Join(k.Dir, "node.key"), "EC PRIVATE KEY", keyDER)
}

func writePEMToFile(t *testing.T, path, typ string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pem.Encode(f, &pem.Block{Type: typ, Bytes: data})
}

func TestCertRenewal_ComputeRenewAt(t *testing.T) {
	kit := newTestCertKit(t)

	// Cert valid for 72 hours, starting now
	notBefore := time.Now()
	notAfter := notBefore.Add(72 * time.Hour)
	kit.issueNodeCert(t, "hestia", notBefore, notAfter)

	cr, err := NewCertRenewal(CertRenewalConfig{
		CertFile: filepath.Join(kit.Dir, "node.crt"),
		KeyFile:  filepath.Join(kit.Dir, "node.key"),
		CAFile:   filepath.Join(kit.Dir, "ca-chain.crt"),
		NodeName: "hestia",
	})
	if err != nil {
		t.Fatalf("NewCertRenewal: %v", err)
	}

	renewAt := cr.RenewAt()
	expected := notBefore.Add(48 * time.Hour) // 2/3 of 72h
	diff := renewAt.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("renewAt = %v, want ~%v (diff: %v)", renewAt, expected, diff)
	}
}

func TestCertRenewal_SkipsWhenNotDue(t *testing.T) {
	kit := newTestCertKit(t)

	// Cert valid for 72 hours starting now — renewal at 48h, not due yet
	kit.issueNodeCert(t, "hestia", time.Now(), time.Now().Add(72*time.Hour))

	cr, err := NewCertRenewal(CertRenewalConfig{
		CertFile: filepath.Join(kit.Dir, "node.crt"),
		KeyFile:  filepath.Join(kit.Dir, "node.key"),
		CAFile:   filepath.Join(kit.Dir, "ca-chain.crt"),
		NodeName: "hestia",
	})
	if err != nil {
		t.Fatalf("NewCertRenewal: %v", err)
	}

	attempted := cr.CheckAndRenew()
	if attempted {
		t.Error("CheckAndRenew should not attempt renewal when cert is fresh")
	}
}

func TestCertRenewal_RenewsFromPeer(t *testing.T) {
	kit := newTestCertKit(t)

	// Cert that's already past 2/3 of its lifetime
	kit.issueNodeCert(t, "hestia", time.Now().Add(-50*time.Hour), time.Now().Add(22*time.Hour))

	// Fake adyton server that returns a renewed cert
	fakePKI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"certificate":   "-----BEGIN CERTIFICATE-----\nrenewed\n-----END CERTIFICATE-----",
			"private_key":   "-----BEGIN EC PRIVATE KEY-----\nnew-key\n-----END EC PRIVATE KEY-----",
			"ca_chain":      "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
			"serial_number": "dd:ee:ff",
			"expiration":    time.Now().Add(72 * time.Hour).Unix(),
		})
	}))
	defer fakePKI.Close()

	adytonClient := node.New("adyton", fakePKI.Listener.Addr().String(), "tok")

	cr, err := NewCertRenewal(CertRenewalConfig{
		CertFile: filepath.Join(kit.Dir, "node.crt"),
		KeyFile:  filepath.Join(kit.Dir, "node.key"),
		CAFile:   filepath.Join(kit.Dir, "ca-chain.crt"),
		NodeName: "hestia",
		Adyton:   adytonClient,
	})
	if err != nil {
		t.Fatalf("NewCertRenewal: %v", err)
	}

	attempted := cr.CheckAndRenew()
	if !attempted {
		t.Fatal("CheckAndRenew should attempt renewal when cert is past 2/3 lifetime")
	}

	// Verify cert was written to disk
	data, err := os.ReadFile(filepath.Join(kit.Dir, "node.crt"))
	if err != nil {
		t.Fatalf("reading renewed cert: %v", err)
	}
	if string(data) != "-----BEGIN CERTIFICATE-----\nrenewed\n-----END CERTIFICATE-----\n" {
		t.Errorf("cert content = %q", string(data))
	}
}

func TestCertRenewal_RenewsLocal(t *testing.T) {
	kit := newTestCertKit(t)

	// Cert that's already past 2/3 of its lifetime
	kit.issueNodeCert(t, "adyton", time.Now().Add(-50*time.Hour), time.Now().Add(22*time.Hour))

	// Fake OpenBao PKI endpoint
	pkiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki_lamina/issue/node" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"certificate":   "-----BEGIN CERTIFICATE-----\nlocal-renewed\n-----END CERTIFICATE-----",
				"private_key":   "-----BEGIN EC PRIVATE KEY-----\nlocal-key\n-----END EC PRIVATE KEY-----",
				"ca_chain":      []string{"-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"},
				"serial_number": "11:22:33",
				"expiration":    time.Now().Add(72 * time.Hour).Unix(),
			},
		})
	}))
	defer pkiSrv.Close()

	baoStore := keychain.NewBaoStore(pkiSrv.URL, "test-token", "secret")
	pkiIssuer := keychain.NewBaoPKIIssuer(baoStore, "pki_lamina")

	cr, err := NewCertRenewal(CertRenewalConfig{
		CertFile:  filepath.Join(kit.Dir, "node.crt"),
		KeyFile:   filepath.Join(kit.Dir, "node.key"),
		CAFile:    filepath.Join(kit.Dir, "ca-chain.crt"),
		NodeName:  "adyton",
		PKIIssuer: pkiIssuer,
	})
	if err != nil {
		t.Fatalf("NewCertRenewal: %v", err)
	}

	attempted := cr.CheckAndRenew()
	if !attempted {
		t.Fatal("CheckAndRenew should attempt local renewal")
	}

	data, err := os.ReadFile(filepath.Join(kit.Dir, "node.crt"))
	if err != nil {
		t.Fatalf("reading renewed cert: %v", err)
	}
	if string(data) != "-----BEGIN CERTIFICATE-----\nlocal-renewed\n-----END CERTIFICATE-----\n" {
		t.Errorf("cert content = %q", string(data))
	}
}
