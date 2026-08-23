package main

import "testing"

func TestDetectWord(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// Non-Latin scripts
		{"Chinese", "你好世界", "喵"},
		{"Japanese kana (hiragana)", "こんにちは", "ニャー"},
		{"Japanese kana (katakana)", "コンニチハ", "ニャー"},
		{"Japanese mixed kanji+kana -> Japanese (kana precedence)", "日本語が分かります", "ニャー"},
		{"Pure kanji -> Chinese fallback", "日本語", "喵"},
		{"Korean", "안녕하세요", "야옹"},
		{"Russian", "Привет", "Мяу"},
		{"Thai", "สวัสดี", "เหมียว"},
		{"Greek", "Γεια σου", "Νιάου"},
		{"Arabic", "مرحبا", "مياو"},
		{"Hebrew", "שלום", "מיאו"},
		{"Hindi", "नमस्ते", "म्याऊ"},

		// Latin heuristics
		{"English plain ASCII", "hello world", "Meow"},
		{"empty input", "", "Meow"},
		{"French (ç)", "bonjour ça va", "Miaou"},
		{"French (ç in question)", "Comment ça va?", "Miaou"},
		{"Spanish (ñ)", "¿hola niño?", "Miau"},
		{"German (ß)", "straße", "Miau"},
		{"Portuguese coração (ç+ã -> Miau precedence)", "coração", "Miau"},
		{"Polish (ł)", "Łódź", "Miau"},
		{"Czech (ř)", "Dvořák", "Miau"},
		{"only acute é (not a marker) -> Meow", "café", "Meow"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectWord(c.in); got != c.want {
				t.Errorf("detectWord(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
