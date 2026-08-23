package main

import "net/http"

// modelIDs is the single source of truth for /v1/models. Edit this slice to
// change the catalog.
var modelIDs = []string{
	// GLM
	"glm-5.2", "glm-5.3",
	// GPT
	"gpt-5.5", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna",
	// Claude (全系列)
	"claude-fable-5", "claude-opus-5", "claude-sonnet-5",
	"claude-haiku-4-5-20251001", "claude-3-5-sonnet-20241022",
	"claude-3-5-haiku-20241022", "claude-3-opus-20240229",
	// Kimi
	"kimi-3",
	// DeepSeek
	"deepseek-v4-flash", "deepseek-v4-pro", "deepseek-v4-flash-vision-exp",
}

// handleModels returns the model catalog in the OpenAI list shape.
// GET /v1/models — always public.
func handleModels(w http.ResponseWriter, r *http.Request) {
	type modelObj struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	out := struct {
		Object string     `json:"object"`
		Data   []modelObj `json:"data"`
	}{
		Object: "list",
		Data:   make([]modelObj, len(modelIDs)),
	}
	for i, id := range modelIDs {
		out.Data[i] = modelObj{ID: id, Object: "model", Created: 1700000000, OwnedBy: "meow"}
	}
	writeJSON(w, http.StatusOK, out)
}
