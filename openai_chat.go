package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// chatReq is the subset of the OpenAI chat-completions request the mock needs.
type chatReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// handleChatCompletions implements POST /v1/chat/completions (OpenAI).
func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{"message": "invalid request body", "type": "invalid_request_error"},
		})
		return
	}
	if req.Model == "" {
		req.Model = "gpt-5.5"
	}

	userText := extractUserText(req.Messages)
	word := detectWord(userText)
	words := generateMeows(word, cfg.MeowMin, cfg.MeowMax)
	promptTokens := countWords(userText)
	completionTokens := len(words)

	if req.Stream {
		streamChatCompletion(w, req.Model, words)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      genID("chatcmpl-"),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": strings.Join(words, " ")},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      promptTokens + completionTokens,
		},
	})
}

// streamChatCompletion emits the OpenAI chat-completion SSE stream: a role
// chunk, one content chunk per word, a final stop chunk, then [DONE].
func streamChatCompletion(w http.ResponseWriter, model string, words []string) {
	setSSEHeaders(w)
	id := genID("chatcmpl-")
	created := time.Now().Unix()

	chunk := func(delta map[string]any, finish any) string {
		return mustJSON(map[string]any{
			"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
			"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finish}},
		})
	}

	writeSSE(w, "", chunk(map[string]any{"role": "assistant"}, nil))
	for _, word := range words {
		writeSSE(w, "", chunk(map[string]any{"content": word + " "}, nil))
		maybeDelay(cfg.StreamDelayMS)
	}
	writeSSE(w, "", chunk(map[string]any{}, "stop"))
	writeSSE(w, "", "[DONE]")
}
