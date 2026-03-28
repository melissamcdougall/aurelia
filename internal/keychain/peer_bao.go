package keychain

import "fmt"

// SecretReader can read secrets by key from a remote source.
type SecretReader interface {
	GetSecret(key string) (string, error)
}

// PeerBaoStore implements Store by proxying secret reads through
// a remote aurelia daemon's API.
//
// Write operations (Set, Delete) are not supported.
type PeerBaoStore struct {
	reader SecretReader
}

// NewPeerBaoStore creates a store that reads secrets from a peer aurelia daemon.
func NewPeerBaoStore(reader SecretReader) *PeerBaoStore {
	return &PeerBaoStore{reader: reader}
}

func (p *PeerBaoStore) Get(key string) (string, error) {
	return p.reader.GetSecret(key)
}

func (p *PeerBaoStore) List() ([]string, error) {
	return nil, fmt.Errorf("peer secret store does not support list")
}

func (p *PeerBaoStore) Set(key, value string) error {
	return fmt.Errorf("peer secret store is read-only")
}

func (p *PeerBaoStore) Delete(key string) error {
	return fmt.Errorf("peer secret store is read-only")
}
