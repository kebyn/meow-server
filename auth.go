package main

import (
	"net/http"
	"strings"
)

// keyFromRequest extracts the API key using the header convention of the given
// provider ("openai" → Bearer, "anthropic" → x-api-key).
func keyFromRequest(r *http.Request, provider string) string {
	if provider == "anthropic" {
		return r.Header.Get("x-api-key")
	}
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

// requireAuth wraps next with key validation. When auth is disabled the wrapped
// handler runs untouched. On failure a provider-native 401 is written and next
// is not called.
func requireAuth(cfg Config, provider string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.AuthEnabled {
			next(w, r)
			return
		}
		if !keyValid(cfg, keyFromRequest(r, provider)) {
			writeAuthError(w, provider)
			return
		}
		next(w, r)
	}
}

// keyValid accepts a key that matches a configured VALID_KEYS entry, or — when
// none are configured — any key starting with KEY_PREFIX (default "sk-").
func keyValid(cfg Config, key string) bool {
	if key == "" {
		return false
	}
	if len(cfg.ValidKeys) > 0 {
		return cfg.ValidKeys[key]
	}
	return strings.HasPrefix(key, cfg.KeyPrefix)
}

// writeAuthError emits a 401 in the provider's native error shape so that
// provider-specific clients parse it correctly.
func writeAuthError(w http.ResponseWriter, provider string) {
	if provider == "anthropic" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"error": map[string]any{
			"message": "invalid api key",
			"type":    "invalid_request_error",
			"code":    "invalid_api_key",
		},
	})
}
