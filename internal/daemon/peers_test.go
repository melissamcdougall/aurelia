package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benaskins/aurelia/internal/config"
	"github.com/benaskins/aurelia/internal/node"
)

func TestWithPeersCreatesClients(t *testing.T) {
	d := NewDaemon(t.TempDir(), WithPeers([]*node.Client{
		node.New("limen", "limen.local:9090", "tok"),
	}))

	if len(d.peers) != 1 {
		t.Fatalf("len(peers) = %d, want 1", len(d.peers))
	}
	if d.peers["limen"] == nil {
		t.Error("expected peer 'limen' to exist")
	}
}

func TestPeerStatesReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := node.New("test-peer", srv.Listener.Addr().String(), "tok")
	d := NewDaemon(t.TempDir(), WithPeers([]*node.Client{c}))
	d.peerStatus["test-peer"] = true

	states := d.PeerStates()
	if len(states) != 1 {
		t.Fatalf("len = %d, want 1", len(states))
	}
	if !states["test-peer"] {
		t.Error("expected test-peer to be reachable")
	}
}

func TestPeerStatesUnreachable(t *testing.T) {
	c := node.New("dead-peer", "127.0.0.1:1", "tok")
	d := NewDaemon(t.TempDir(), WithPeers([]*node.Client{c}))
	d.peerStatus["dead-peer"] = false

	states := d.PeerStates()
	if states["dead-peer"] {
		t.Error("expected dead-peer to be unreachable")
	}
}

func TestBuildPeersFromConfig(t *testing.T) {
	dir := t.TempDir()
	tokenFile := dir + "/token"
	if err := writeTokenFile(tokenFile, "file-tok"); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		NodeName: "aurelia",
		Nodes: []config.Node{
			{Name: "aurelia", Addr: "aurelia.local:9090", Token: "self-tok"},
			{Name: "limen", Addr: "limen.local:9090", TokenFile: tokenFile},
		},
	}

	peers := BuildPeers(cfg, nil)
	if len(peers) != 1 {
		t.Fatalf("len(peers) = %d, want 1 (self excluded)", len(peers))
	}
	if peers[0].Name != "limen" {
		t.Errorf("Name = %q, want limen", peers[0].Name)
	}
}

func TestPeerLivenessUpdatesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := node.New("live-peer", srv.Listener.Addr().String(), "tok")
	d := NewDaemon(t.TempDir(), WithPeers([]*node.Client{c}))

	// Run one liveness check
	d.checkPeerLiveness()

	states := d.PeerStates()
	if !states["live-peer"] {
		t.Error("expected live-peer to be reachable after liveness check")
	}
}

func writeTokenFile(path, token string) error {
	return os.WriteFile(path, []byte(token), 0600)
}
