# meow-server 🐱

[English](README.md) | **中文**

一个用 Go 编写的轻量级 LLM 模拟接口服务器（仅使用标准库）。它实现了三种常见的
LLM HTTP 接口，但不返回真实补全，而是随机返回**若干个"喵"或"Meow"**——根据用户输入
语言选择词汇（中文 → 喵，其他语言 → Meow）。可作为假后端，零 API 成本地测试 LLM
客户端。

## 功能特性

- **三种接口**，每种都支持非流式 JSON 与 SSE 流式（`stream=true`）：
  - `POST /v1/chat/completions` —— OpenAI Chat Completions
  - `POST /v1/responses` —— OpenAI Responses API
  - `POST /v1/messages` —— Anthropic Messages API
- `GET /v1/models` —— 模型列表（始终公开）
- **鉴权**：默认开启，校验以 `sk-` 开头的 key；也可关闭。还支持精确 key 白名单。
  401 错误按各厂商原生错误格式返回。
- **按语言选词**：将用户输入语言映射到对应的猫叫拟声词——中文 喵、日文 ニャー、
  韩文 야옹、俄文 Мяу、法文 Miaou……见[按语言映射](#按语言映射)。
- 内置宽松 CORS，浏览器客户端可直接访问。
- 无外部依赖——`go build` 即可运行。

## 快速开始

```bash
go build .
./meow-server
# 2026/.../... meow-server listening on :8080 (auth=true, models=17, meows=1-20)
```

本地快速测试可关闭鉴权：

```bash
AUTH_ENABLED=false ./meow-server
```

## 配置

所有配置来自环境变量。

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | 监听端口。 |
| `AUTH_ENABLED` | `true` | 是否对三种补全接口校验 key。 |
| `KEY_PREFIX` | `sk-` | 未设置 `VALID_KEYS` 时，key 必须以该前缀开头。 |
| `VALID_KEYS` | *(空)* | 逗号分隔的精确 key 列表。设置后 key 必须完全匹配其中之一（覆盖前缀校验）。 |
| `MEOW_MIN` / `MEOW_MAX` | `1` / `20` | 返回喵/Meow 数量的随机区间。 |
| `STREAM_DELAY_MS` | `0` | 流式输出时每个词之间的延迟（让 SSE 逐词可见）。 |

## 接口

### `GET /v1/models`

```bash
curl -s http://localhost:8080/v1/models | jq '.data[].id'
```

### `POST /v1/chat/completions`（OpenAI）

中文输入 → `喵`；日文 → `ニャー`；法文 → `Miaou`；英文 → `Meow`。见
[按语言映射](#按语言映射)。

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.5","messages":[{"role":"user","content":"你好"}]}'
```

流式：

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.5","stream":true,"messages":[{"role":"user","content":"hello"}]}'
# 帧：role 帧 → content 帧（每个词一帧）→ stop 帧 → data: [DONE]
```

### `POST /v1/responses`（OpenAI Responses API）

```bash
curl -s http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.6-luna","input":"hello"}'
```

流式依次发送 `response.created` → `response.output_item.added` →
`response.content_part.added` → `response.output_text.delta`（每个词）→
`response.output_text.done` → `response.content_part.done` →
`response.output_item.done` → `response.completed`。请求中加 `"stream":true`。

### `POST /v1/messages`（Anthropic）

Anthropic 使用 `x-api-key` 头鉴权。

```bash
curl -s http://localhost:8080/v1/messages \
  -H "x-api-key: sk-test" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-opus-5","max_tokens":64,"messages":[{"role":"user","content":"你好"}]}'
```

流式依次发送 `message_start` → `content_block_start` → `content_block_delta`
（每个词）→ `content_block_stop` → `message_delta` → `message_stop`。请求中加
`"stream":true`。

## 按语言映射

返回的词根据用户输入语言选择。非拉丁文种按 Unicode 范围精确检测；拉丁语种按特征性
变音符号做尽力而为的猜测。

| 文字 / 信号 | 语言 | 拟声词 |
|---|---|---|
| 谚文（Hangul） | 韩文 | `야옹` |
| 假名（kana） | 日文 | `ニャー` |
| CJK 汉字 | 中文 | `喵` |
| 西里尔文 | 俄文等 | `Мяу` |
| 泰文 | 泰文 | `เหมียว` |
| 希腊文 | 希腊文 | `Νιάου` |
| 阿拉伯文 | 阿拉伯文 | `مياو` |
| 希伯来文 | 希伯来文 | `מיאו` |
| 天城文 | 印地文 | `म्याऊ` |
| 拉丁文——法语标记（`ç è à ù …`） | 法文 | `Miaou` |
| 拉丁文——其他变音符号（`ñ ß ã ł ř …`） | 西/德/葡/波兰/捷克…… | `Miau` |
| 拉丁文——纯 ASCII | 英文（默认） | `Meow` |

说明：假名优先于 CJK 检测，故含假名的日文会识别为日文；只有纯汉字的输入才会归为中文。
常见尖音符号（é á í ó ú）因在多种语言中通用而未作为标记，因此 `café` → `Meow`。
修改 `meow.go` 的 `scriptSounds` 切片即可调整映射。

## 鉴权

| 模式 | 行为 |
|---|---|
| 默认（`AUTH_ENABLED=true`） | OpenAI 接口读取 `Authorization: Bearer <key>`；Anthropic 读取 `x-api-key`。key 必须以 `sk-` 开头（或匹配 `VALID_KEYS` 中的一项）。 |
| `AUTH_ENABLED=false` | 不校验 key。 |
| `VALID_KEYS=sk-secret,sk-prod` | 仅接受这些精确 key；通用的 `sk-anything` 会被拒绝。 |

## 模型

```
glm-5.2  glm-5.3
gpt-5.5  gpt-5.6-sol  gpt-5.6-terra  gpt-5.6-luna
claude-fable-5  claude-opus-5  claude-sonnet-5  claude-haiku-4-5-20251001
claude-3-5-sonnet-20241022  claude-3-5-haiku-20241022  claude-3-opus-20240229
kimi-3
deepseek-v4-flash  deepseek-v4-pro  deepseek-v4-flash-vision-exp
```

修改 `models.go` 中的 `modelIDs` 切片即可调整列表。服务器接受请求中的任意模型字符串
并原样回显（它是个模拟器）。

## 项目结构

```
main.go              配置加载、路由、CORS、启动服务
config.go            从环境变量构造 Config
auth.go              requireAuth 中间件 + 厂商原生 401
meow.go              中文检测、喵生成、SSE/JSON 工具
models.go            模型列表 + GET /v1/models
openai_chat.go       POST /v1/chat/completions （流式 + 非流式）
openai_responses.go  POST /v1/responses         （流式 + 非流式）
anthropic.go         POST /v1/messages          （流式 + 非流式）
```

## 可复现构建 (Reproducible Builds)

二进制为逐字节可复现构建——相同源码始终产出相同产物（sha256 一致），无论本地或 Docker 构建。

- 本地可复现构建：`make build`
- 验证确定性（两次隔离缓存构建，sha256 比对）：`make verify-repro`
- Docker 镜像：`make docker`（与本地二进制逐字节一致）

工具链与 flag 已固定（`go1.24.4`、`-trimpath -buildvcs=false -ldflags='-s -w'`、
`CGO_ENABLED=0`、`GOOS=linux GOARCH=amd64 GOAMD64=v1`、`SOURCE_DATE_EPOCH=0`），
并在 `Makefile` 与 `Dockerfile` 间完全镜像。比对本地与 Docker 产物：

    make build && make docker
    cid=$(docker create meow-server) && docker cp "$cid:/meow-server" ./meow-server.docker && docker rm "$cid" >/dev/null
    cmp ./meow-server ./meow-server.docker && echo "local == docker: OK"

## 说明

- 纯模拟——不发起真实 LLM 调用，不代理上游。
- 仅文本；不支持工具/函数调用或多模态内容。
- 无 TLS；生产环境请在反向代理后运行。
