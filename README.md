# meow-server 🐱

**English** | [中文](README.zh.md)

A lightweight mock LLM API server written in Go (standard library only). It
implements the three common LLM HTTP interfaces and returns **a random number of
"喵" or "Meow"** instead of real completions — choosing the word by the user's
input language (Chinese → 喵, anything else → Meow). Drop it in as a fake
backend for prototyping and testing LLM clients with zero API spend.

## Features

- **Three endpoints**, each supporting both non-streaming JSON and SSE (`stream=true`):
  - `POST /v1/chat/completions` — OpenAI Chat Completions
  - `POST /v1/responses` — OpenAI Responses API
  - `POST /v1/messages` — Anthropic Messages API
- `GET /v1/models` — model catalog (always public)
- **Auth**: validate keys starting with `sk-` (on by default), or disable it.
  Also supports an exact-key allowlist. 401 errors are returned in each
  provider's native error shape.
- **Language-aware meows**: maps the user's input language to its cat sound —
  Chinese 喵, Japanese ニャー, Korean 야옹, Russian Мяу, French Miaou, …
  See [Sounds by language](#sounds-by-language).
- Permissive CORS, so browser-based clients work directly.
- No external dependencies — `go build` and run.

## Quick start

```bash
go build .
./meow-server
# 2026/.../... meow-server listening on :8080 (auth=true, models=17, meows=1-20)
```

Run without auth for quick local testing:

```bash
AUTH_ENABLED=false ./meow-server
```

## Configuration

All settings come from environment variables.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Listen port. |
| `AUTH_ENABLED` | `true` | Require a key on the three completion endpoints. |
| `KEY_PREFIX` | `sk-` | Required key prefix when `VALID_KEYS` is unset. |
| `VALID_KEYS` | *(empty)* | Comma-separated exact keys. When set, a key must match one exactly (overrides the prefix check). |
| `MEOW_MIN` / `MEOW_MAX` | `1` / `20` | Random count range of returned meows. |
| `STREAM_DELAY_MS` | `0` | Delay between streamed words (makes SSE visibly token-by-token). |

## Endpoints

### `GET /v1/models`

```bash
curl -s http://localhost:8080/v1/models | jq '.data[].id'
```

### `POST /v1/chat/completions` (OpenAI)

Chinese input → `喵`; Japanese → `ニャー`; French → `Miaou`; English → `Meow`.
See [Sounds by language](#sounds-by-language).

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.5","messages":[{"role":"user","content":"你好"}]}'
```

Streaming:

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.5","stream":true,"messages":[{"role":"user","content":"hello"}]}'
# frames: role chunk → content chunks (one per word) → stop chunk → data: [DONE]
```

### `POST /v1/responses` (OpenAI Responses API)

```bash
curl -s http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.6-luna","input":"hello"}'
```

Streaming sends `response.created` → `response.output_item.added` →
`response.content_part.added` → `response.output_text.delta` (per word) →
`response.output_text.done` → `response.content_part.done` →
`response.output_item.done` → `response.completed`. Add `"stream":true`.

### `POST /v1/messages` (Anthropic)

Anthropic auth uses the `x-api-key` header.

```bash
curl -s http://localhost:8080/v1/messages \
  -H "x-api-key: sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-opus-5","max_tokens":64,"messages":[{"role":"user","content":"你好"}]}'
```

Streaming sends `message_start` → `content_block_start` → `content_block_delta`
(per word) → `content_block_stop` → `message_delta` → `message_stop`. Add
`"stream":true`.

## Sounds by language

The returned word is chosen by the user's input language. Non-Latin scripts are
detected exactly by Unicode ranges; Latin languages are guessed from
distinctive diacritics (best-effort).

| Script / signal | Language | Sound |
|---|---|---|
| Hangul | Korean | `야옹` |
| Kana (hiragana/katakana) | Japanese | `ニャー` |
| CJK ideographs | Chinese | `喵` |
| Cyrillic | Russian etc. | `Мяу` |
| Thai | Thai | `เหมียว` |
| Greek | Greek | `Νιάου` |
| Arabic | Arabic | `مياو` |
| Hebrew | Hebrew | `מיאו` |
| Devanagari | Hindi | `म्याऊ` |
| Latin — French markers (`ç è à ù …`) | French | `Miaou` |
| Latin — other diacritics (`ñ ß ã ł ř …`) | Spanish / German / Portuguese / Polish / Czech … | `Miau` |
| Latin — plain ASCII | English (default) | `Meow` |

Notes: kana is checked before CJK, so mixed Japanese text resolves to Japanese;
only pure-kanji input falls through to Chinese. Common acute accents (`é á í ó ú`)
are not markers because they're shared across many languages, so `café` → `Meow`.
Edit the `scriptSounds` slice in `meow.go` to change the catalog.

## Auth

| Mode | Behavior |
|---|---|
| Default (`AUTH_ENABLED=true`) | OpenAI endpoints read `Authorization: Bearer <key>`; Anthropic reads `x-api-key`. The key must start with `sk-` (or match a `VALID_KEYS` entry). |
| `AUTH_ENABLED=false` | No key check. |
| `VALID_KEYS=sk-secret,sk-prod` | Only those exact keys are accepted; a generic `sk-anything` is rejected. |

## Models

```
glm-5.2  glm-5.3
gpt-5.5  gpt-5.6-sol  gpt-5.6-terra  gpt-5.6-luna
claude-fable-5  claude-opus-5  claude-sonnet-5  claude-haiku-4-5-20251001
claude-3-5-sonnet-20241022  claude-3-5-haiku-20241022  claude-3-opus-20240229
kimi-3
deepseek-v4-flash  deepseek-v4-pro  deepseek-v4-flash-vision-exp
```

Edit the `modelIDs` slice in `models.go` to change the catalog. The server
accepts any model string in requests and echoes it back (it's a mock).

## Project layout

```
main.go              config load, routing, CORS, server start
config.go            Config struct from env
auth.go              requireAuth middleware + provider-shaped 401
meow.go              CJK detection, meow generation, SSE/JSON helpers
models.go            model catalog + GET /v1/models
openai_chat.go       POST /v1/chat/completions  (stream + non-stream)
openai_responses.go  POST /v1/responses         (stream + non-stream)
anthropic.go         POST /v1/messages          (stream + non-stream)
```

## Reproducible Builds

The binary is built byte-for-byte reproducible — identical source always yields
identical output (same sha256), whether built locally or via Docker.

- Local reproducible build: `make build`
- Prove determinism (two isolated-cache builds, sha256 match): `make verify-repro`
- Docker image: `make docker` (byte-identical to the local binary)

Toolchain and flags are pinned (`go1.24.4`, `-trimpath -buildvcs=false -ldflags='-s -w'`,
`CGO_ENABLED=0`, `GOOS=linux GOARCH=amd64 GOAMD64=v1`, `SOURCE_DATE_EPOCH=0`) and mirrored
between the `Makefile` and `Dockerfile`. Compare local vs Docker output:

    make build && make docker
    cid=$(docker create meow-server) && docker cp "$cid:/meow-server" ./meow-server.docker && docker rm "$cid" >/dev/null
    cmp ./meow-server ./meow-server.docker && echo "local == docker: OK"

## Notes

- Pure mock — no real LLM calls, no upstream proxying.
- Text only; no tool/function calling or multimodal content.
- No TLS; run behind a reverse proxy in production.
