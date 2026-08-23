package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// responsesReq is the subset of the OpenAI Responses API request the mock
// needs. Input may be a string or an array of message items.
type responsesReq struct {
	Model  string `json:"model"`
	Input  any    `json:"input"`
	Stream bool   `json:"stream"`
}

// handleResponses implements POST /v1/responses (OpenAI Responses API).
func handleResponses(w http.ResponseWriter, r *http.Request) {
	var req responsesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{"message": "invalid request body", "type": "invalid_request_error"},
		})
		return
	}
	if req.Model == "" {
		req.Model = "gpt-5.5"
	}

	userText := responsesInputText(req.Input)
	word := detectWord(userText)
	words := generateMeows(word, cfg.MeowMin, cfg.MeowMax)
	promptTokens := countWords(userText)
	completionTokens := len(words)
	content := strings.Join(words, " ")

	respID := genID("resp_")
	msgID := genID("msg_")

	if req.Stream {
		streamResponses(w, req.Model, words, content, promptTokens, completionTokens, respID, msgID)
		return
	}
	writeJSON(w, http.StatusOK, buildResponseObject(req.Model, content, promptTokens, completionTokens, respID, msgID))
}

// responsesInputText extracts user-visible text from the Responses `input`
// field, which is either a string or an array of message items.
func responsesInputText(in any) string {
	switch v := in.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, item := range v {
			if mp, ok := item.(map[string]any); ok {
				b.WriteString(contentText(mp["content"]))
			}
		}
		return b.String()
	}
	return ""
}

// buildResponseObject constructs the OpenAI Responses non-streaming response.
func buildResponseObject(model, content string, promptTokens, completionTokens int, respID, msgID string) map[string]any {
	return map[string]any{
		"id":         respID,
		"object":     "response",
		"created_at": time.Now().Unix(),
		"status":     "completed",
		"model":      model,
		"output": []map[string]any{{
			"type":   "message",
			"id":     msgID,
			"status": "completed",
			"role":   "assistant",
			"content": []map[string]any{{
				"type":        "output_text",
				"text":        content,
				"annotations": []any{},
			}},
		}},
		"usage": map[string]any{
			"input_tokens":  promptTokens,
			"output_tokens": completionTokens,
			"total_tokens":  promptTokens + completionTokens,
		},
	}
}

// streamResponses emits the Responses SSE event sequence (verified against the
// openai-python ResponseStreamEvent union): created → output_item.added →
// content_part.added → output_text.delta (per word) → output_text.done →
// content_part.done → output_item.done → completed.
func streamResponses(w http.ResponseWriter, model string, words []string, fullContent string, promptTokens, completionTokens int, respID, msgID string) {
	setSSEHeaders(w)
	created := time.Now().Unix()

	partial := map[string]any{
		"id": respID, "object": "response", "created_at": created, "status": "in_progress",
		"model": model, "output": []any{},
		"usage": map[string]any{"input_tokens": promptTokens, "output_tokens": 0, "total_tokens": promptTokens},
	}

	writeSSE(w, "response.created", mustJSON(map[string]any{"type": "response.created", "response": partial}))
	writeSSE(w, "response.output_item.added", mustJSON(map[string]any{
		"type": "response.output_item.added", "output_index": 0,
		"item": map[string]any{"type": "message", "id": msgID, "status": "in_progress", "role": "assistant", "content": []any{}},
	}))
	writeSSE(w, "response.content_part.added", mustJSON(map[string]any{
		"type": "response.content_part.added", "item_id": msgID, "output_index": 0, "content_index": 0,
		"part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}},
	}))

	for _, word := range words {
		writeSSE(w, "response.output_text.delta", mustJSON(map[string]any{
			"type": "response.output_text.delta", "item_id": msgID,
			"output_index": 0, "content_index": 0, "delta": word + " ",
		}))
		maybeDelay(cfg.StreamDelayMS)
	}

	writeSSE(w, "response.output_text.done", mustJSON(map[string]any{
		"type": "response.output_text.done", "item_id": msgID,
		"output_index": 0, "content_index": 0, "text": fullContent,
	}))
	writeSSE(w, "response.content_part.done", mustJSON(map[string]any{
		"type": "response.content_part.done", "item_id": msgID, "output_index": 0, "content_index": 0,
		"part": map[string]any{"type": "output_text", "text": fullContent, "annotations": []any{}},
	}))
	writeSSE(w, "response.output_item.done", mustJSON(map[string]any{
		"type": "response.output_item.done", "output_index": 0,
		"item": map[string]any{
			"type": "message", "id": msgID, "status": "completed", "role": "assistant",
			"content": []map[string]any{{"type": "output_text", "text": fullContent, "annotations": []any{}}},
		},
	}))
	writeSSE(w, "response.completed", mustJSON(map[string]any{
		"type":     "response.completed",
		"response": buildResponseObject(model, fullContent, promptTokens, completionTokens, respID, msgID),
	}))
}
