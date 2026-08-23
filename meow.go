package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// chatMessage is the shared message shape used by the chat-completions and
// messages request bodies. Content may be a plain string or an array of
// content parts, hence any.
type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// isCJK reports whether r is a CJK ideograph (Chinese characters).
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
		(r >= 0xF900 && r <= 0xFAFF) // CJK Compatibility Ideographs
}

func isHangul(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0x1100 && r <= 0x11FF) || // Hangul Jamo
		(r >= 0x3130 && r <= 0x318F) // Hangul Compatibility Jamo
}

func isKana(r rune) bool {
	return (r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) // Katakana
}

func isCyrillic(r rune) bool {
	return (r >= 0x0400 && r <= 0x04FF) || // Cyrillic
		(r >= 0x0500 && r <= 0x052F) // Cyrillic Supplement
}

func isThai(r rune) bool  { return r >= 0x0E00 && r <= 0x0E7F }
func isGreek(r rune) bool { return r >= 0x0370 && r <= 0x03FF }

func isArabic(r rune) bool {
	return (r >= 0x0600 && r <= 0x06FF) || // Arabic
		(r >= 0x0750 && r <= 0x077F) || // Arabic Supplement
		(r >= 0xFB50 && r <= 0xFDFF) || // Arabic Presentation Forms-A
		(r >= 0xFE70 && r <= 0xFEFF) // Arabic Presentation Forms-B
}

func isHebrew(r rune) bool     { return r >= 0x0590 && r <= 0x05FF }
func isDevanagari(r rune) bool { return r >= 0x0900 && r <= 0x097F }

// scriptSound pairs a script detector with the cat onomatopoeia for that
// language. Checked in priority order.
type scriptSound struct {
	match func(rune) bool
	sound string
}

// scriptSounds maps detected scripts to cat sounds. Kana is checked before CJK
// so mixed Japanese text (kanji + kana) resolves to Japanese; only pure-kanji
// input falls through to Chinese (喵). Edit this slice to change the catalog.
var scriptSounds = []scriptSound{
	{isHangul, "야옹"},
	{isKana, "ニャー"},
	{isCJK, "喵"},
	{isCyrillic, "Мяу"},
	{isThai, "เหมียว"},
	{isGreek, "Νιάου"},
	{isArabic, "مياو"},
	{isHebrew, "מיאו"},
	{isDevanagari, "म्याऊ"},
}

// detectWord returns the cat onomatopoeia for the user's input language.
// Non-Latin scripts are detected by Unicode script; Latin languages are
// disambiguated by distinctive diacritics (best-effort), defaulting to "Meow".
func detectWord(text string) string {
	for _, ss := range scriptSounds {
		for _, r := range text {
			if ss.match(r) {
				return ss.sound
			}
		}
	}
	return latinSound(text)
}

// latinSound guesses a cat sound for Latin-script text from distinctive
// diacritics. If any non-French Latin marker is present → "Miau" (this also
// sends Portuguese "coração" to "Miau" despite its ç); else if a French
// marker is present → "Miaou"; else plain ASCII → "Meow". The common acute
// accents (é á í ó ú) are deliberately not markers — shared by many languages.
func latinSound(text string) string {
	french := false
	otherLatin := false
	for _, r := range text {
		switch r {
		// French-leaning markers.
		case 'ç', 'Ç', 'œ', 'Œ', 'æ', 'Æ',
			'è', 'È', 'ê', 'Ê', 'à', 'À', 'ù', 'Ù', 'û', 'Û',
			'î', 'Î', 'ï', 'Ï', 'â', 'Â', 'ô', 'Ô':
			french = true
		// Distinctive markers of other Latin languages:
		// ñ(Spanish) ãõ(Portuguese) ßäöü(German) łężżśćńź(Polish) řčšžďť(Czech/Slovak).
		case 'ñ', 'Ñ', 'ã', 'Ã', 'õ', 'Õ', 'ß', 'ä', 'Ä', 'ö', 'Ö', 'ü', 'Ü',
			'ł', 'Ł', 'ę', 'Ę', 'ż', 'Ż', 'ś', 'Ś', 'ć', 'Ć', 'ń', 'Ń', 'ź', 'Ź',
			'ř', 'Ř', 'č', 'Č', 'š', 'Š', 'ž', 'Ž', 'ď', 'Ď', 'ť', 'Ť':
			otherLatin = true
		}
	}
	switch {
	case otherLatin:
		return "Miau"
	case french:
		return "Miaou"
	default:
		return "Meow"
	}
}

// generateMeows returns a random count (in [min,max]) of copies of word. The
// slice form lets streaming handlers emit one word per delta.
func generateMeows(word string, min, max int) []string {
	n := min
	if max > min {
		n = min + rand.IntN(max-min+1)
	}
	if n < 1 {
		n = 1
	}
	out := make([]string, n)
	for i := range out {
		out[i] = word
	}
	return out
}

// countWords is a crude token estimate used for usage stats.
func countWords(s string) int {
	return len(strings.Fields(s))
}

// contentText extracts text from a message content field, which may be a plain
// string or an array of content parts (each carrying a "text" field). This
// handles OpenAI's "text"/"output_text"/"input_text" parts and Anthropic
// "text" blocks uniformly.
func contentText(c any) string {
	switch v := c.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, item := range v {
			if mp, ok := item.(map[string]any); ok {
				if t, ok := mp["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	}
	return ""
}

// extractUserText returns the text of the last user message.
func extractUserText(messages []chatMessage) string {
	var last string
	for _, m := range messages {
		if m.Role == "user" {
			last = contentText(m.Content)
		}
	}
	return last
}

// setSSEHeaders primes the response for Server-Sent Events streaming.
func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

// writeSSE writes one SSE frame. If event is empty, no "event:" line is emitted
// (the OpenAI chat-completions streaming format uses data-only frames).
func writeSSE(w http.ResponseWriter, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// writeJSON writes a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// mustJSON marshals v to a string. Errors are impossible for our map inputs.
func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// genID returns "<prefix><random digits>".
func genID(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, rand.Int64N(1_000_000_000))
}

// maybeDelay sleeps for ms milliseconds if ms > 0, pacing streamed output.
func maybeDelay(ms int) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}
