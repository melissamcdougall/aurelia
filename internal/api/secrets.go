package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/benaskins/aurelia/internal/keychain"
)

// SetSecretCache configures the cached secret store for serving
// secret lookups over the local unix socket.
func (s *Server) SetSecretCache(cache *keychain.CachedStore) {
	s.secretCache = cache
}

func (s *Server) secretGet(w http.ResponseWriter, r *http.Request) {
	if s.secretCache == nil {
		http.Error(w, `{"error":"secret cache not configured"}`, http.StatusServiceUnavailable)
		return
	}

	key := r.PathValue("key")
	val, err := s.secretCache.Get(key)
	if err != nil {
		if errors.Is(err, keychain.ErrNotFound) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		s.logger.Error("secret get failed", "key", key, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"value": val})
}
