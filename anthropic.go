package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// anthropicReq is the subset of the Anthropic Messages request the mock needs.
type anthropicReq struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	System    any           `json:"system"`
}

// handleAnthropicMessages implements POST /v1/messages (Anthropic).
func handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	var req anthropicReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"type":  "error",
			"error": map[string]any{"type": "invalid_request_error", "message": "invalid request body"},
		})
		return
	}
	if req.Model == "" {
		req.Model = "claude-sonnet-5"
	}

	userText := extractUserText(req.Messages)
	word := detectWord(userText)
	words := generateMeows(word, cfg.MeowMin, cfg.MeowMax)
	promptTokens := countWords(userText)
	completionTokens := len(words)
	content := strings.Join(words, " ")

	if req.Stream {
		streamAnthropic(w, req.Model, words, content, promptTokens, completionTokens)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":            genID("msg_"),
		"type":          "message",
		"role":          "assistant",
		"model":         req.Model,
		"content":       []map[string]any{{"type": "text", "text": content}},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  promptTokens,
			"output_tokens": completionTokens,
		},
	})
}

// streamAnthropic emits the Anthropic Messages SSE sequence: message_start →
// content_block_start → content_block_delta (per word) → content_block_stop →
// message_delta → message_stop.
func streamAnthropic(w http.ResponseWriter, model string, words []string, fullContent string, promptTokens, completionTokens int) {
	setSSEHeaders(w)
	msgID := genID("msg_")

	writeSSE(w, "message_start", mustJSON(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id": msgID, "type": "message", "role": "assistant", "model": model,
			"content": []any{}, "stop_reason": nil, "stop_sequence": nil,
			"usage": map[string]any{"input_tokens": promptTokens, "output_tokens": 0},
		},
	}))

	writeSSE(w, "content_block_start", mustJSON(map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "text", "text": ""},
	}))

	for _, word := range words {
		writeSSE(w, "content_block_delta", mustJSON(map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "text_delta", "text": word + " "},
		}))
		maybeDelay(cfg.StreamDelayMS)
	}

	writeSSE(w, "content_block_stop", mustJSON(map[string]any{"type": "content_block_stop", "index": 0}))

	writeSSE(w, "message_delta", mustJSON(map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]any{"output_tokens": completionTokens},
	}))

	writeSSE(w, "message_stop", mustJSON(map[string]any{"type": "message_stop"}))
}
