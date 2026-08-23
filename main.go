package main

import (
	"log"
	"net/http"
)

// cfg is the process-wide configuration, loaded once at startup.
var cfg Config

func main() {
	cfg = loadConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", handleModels)
	mux.HandleFunc("POST /v1/chat/completions", requireAuth(cfg, "openai", handleChatCompletions))
	mux.HandleFunc("POST /v1/responses", requireAuth(cfg, "openai", handleResponses))
	mux.HandleFunc("POST /v1/messages", requireAuth(cfg, "anthropic", handleAnthropicMessages))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	})

	addr := ":" + cfg.Port
	log.Printf("meow-server listening on %s (auth=%v, models=%d, meows=%d-%d)",
		addr, cfg.AuthEnabled, len(modelIDs), cfg.MeowMin, cfg.MeowMax)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

// withCORS adds permissive CORS headers so browser-based clients can hit the
// mock directly, and answers preflight OPTIONS requests.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, x-api-key, anthropic-version")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
