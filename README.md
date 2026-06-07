# perplexity-bridge

<div align="center">

[English](#english) · [中文](#中文) · [日本語](#日本語)

</div>

---

# English

An OpenAI-compatible proxy for Perplexity AI. Uses the Perplexity web API under the hood, exposes a standard `/v1/chat/completions` endpoint.

## What's New vs Upstream

This fork adds the following on top of the original [pplx2api](https://github.com/yushangxiao/pplx2api):

- **Tool calling (function calling)** – the upstream only supports plain chat. This fork adds a meta-request proxy: when `tools` are in the request, it asks Perplexity "which tool to call?", parses the JSON response, and returns OpenAI-compatible `tool_calls`. This enables agentic use (e.g. OpenCode, LangChain, Vercel AI SDK) without Perplexity natively supporting function calling.
- **Updated model list** – 22 validated model mappings covering Claude 4.6, GPT 5.4, Gemini 3.1, Grok 4.1, Sonar, Kimi, DeepSeek R1, o4-mini, and more. Each available with or without `-search` suffix.
- **Provider-prefix stripping** – models can be requested as `pplx2api/claude-4.6-sonnet` (the prefix is stripped automatically), compatible with OpenCode's provider selector.

## Features

- **OpenAI-compatible API** – drop-in replacement for any OpenAI client
- **Tool calling** – meta-request proxy converts user tool definitions into a Perplexity query, parses the response, and returns standard `tool_calls` (no native function-calling needed)
- **Streaming & non-streaming** – full SSE streaming support
- **Image recognition** – send images for analysis
- **Web search** – append `-search` to any model name
- **Thinking models** – access reasoning models, outputs `<think>` tags
- **Multiple accounts** – comma-separated sessions for round-robin / retry
- **Model monitoring** – detects if Perplexity falls back to a different model

## Supported Models

| Friendly Name | Internal Name |
|---|---|
| Claude 4.6 Sonnet | `claude46sonnet` |
| Claude 4.6 Sonnet Think | `claude46sonnetthinking` |
| GPT 5.4 | `gpt54` |
| GPT 5.4 Think | `gpt54_thinking` |
| GPT 5.2 | `gpt52` |
| Claude 4.5 Sonnet | `claude45sonnet` |
| Gemini 3.1 Pro | `gemini31pro_high` |
| Gemini 3 Pro | `gemini3pro` |
| Grok 4.1 | `grok41` |
| Sonar | `sonar` |
| Sonar Pro | `sonar-pro` |
| Sonar Reasoning | `sonar_reasoning` |
| Sonar Reasoning Pro | `sonar-reasoning-pro` |
| Sonar Deep Research | `sonar_deep_research` |
| Kimi | `kimi` |
| Kimi K2 | `kimi-k2` |
| Kimi K2 Think | `kimi_k2_thinking` |
| o4-mini | `o4-mini` |
| GPT 4o | `gpt-4o` |
| GPT 4.1 | `gpt-4.1` |
| Claude 4.0 Sonnet | `claude-4.0-sonnet` |
| DeepSeek R1 | `deepseek-r1` |

Append `-search` to any model for web search mode (e.g. `claude-4.6-sonnet-search`).

## Quick Start

1. Get your `__Secure-next-auth.session-token` cookie from https://perplexity.ai
2. Create `.env`:

```
SESSIONS=<your-cookie>
APIKEY=<your-chosen-key>
ADDRESS=0.0.0.0:8080
IS_INCOGNITO=true
```

3. Run:

```bash
go build -o pplx2api.exe .
pplx2api.exe
```

Or use Docker:

```bash
docker run -d \
  -p 8080:8080 \
  -e SESSIONS=<your-cookie> \
  -e APIKEY=123 \
  ghcr.io/yushangxiao/pplx2api:latest
```

## API

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-4.6-sonnet",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

## Tool Calling

When `tools` are present in the request, the proxy sends a meta-prompt to Perplexity asking which tool to invoke, then returns a standard `tool_calls` response. After the tool result is returned, the second round produces the final answer.

## Configuration

| Variable | Description |
|---|---|
| `SESSIONS` | Perplexity session cookie(s), comma-separated |
| `APIKEY` | API key for authentication |
| `ADDRESS` | Listen address (default `0.0.0.0:8080`) |
| `PROXY` | HTTP proxy URL |
| `IS_INCOGNITO` | Use incognito sessions (default `true`) |
| `IS_MAX_SUBSCRIBE` | Enable Max-tier models (default `false`) |
| `IGNORE_SEARCH_RESULT` | Hide search results in output |

## License

MIT

## Acknowledgements

Based on the original [pplx2api](https://github.com/yushangxiao/pplx2api) by yushangxiao.

---

# 中文

基于 Perplexity 网页接口的 OpenAI 兼容代理。将 Perplexity 的私有 API 转换为标准的 `/v1/chat/completions` 端点，支持工具调用（Tool Calling）、流式输出、识图、联网搜索等功能，可无缝接入 OpenCode、LangChain、Vercel AI SDK 等任意 OpenAI 客户端。

## 与上游项目的差异

本 Fork 在原始 [pplx2api](https://github.com/yushangxiao/pplx2api) 基础上新增：

- **工具调用（Function Calling）** – 上游仅支持普通对话。本 Fork 通过元请求代理实现：当请求中包含 `tools` 时，向 Perplexity 询问"该调用哪个工具"，解析 JSON 响应后返回 OpenAI 兼容的 `tool_calls`，让 Perplexity 也能用于 Agent 场景（OpenCode、LangChain 等）
- **更新模型列表** – 22 个已验证的模型映射，涵盖 Claude 4.6、GPT 5.4、Gemini 3.1、Grok 4.1、Sonar、Kimi、DeepSeek R1、o4-mini 等，每个模型均可附加 `-search` 后缀使用联网搜索
- **提供商前缀剥离** – 支持 `pplx2api/claude-4.6-sonnet` 格式，前缀自动剥离，兼容 OpenCode 的提供商选择器

---

# 日本語

Perplexity の Web API を OpenAI 互換エンドポイントに変換するプロキシです。ツール呼び出し、ストリーミング、画像認識、Web 検索などをサポート。OpenCode、LangChain、Vercel AI SDK などの OpenAI クライアントからそのまま利用できます。

## 元のプロジェクトとの違い

- **ツール呼び出し** – リクエストに `tools` が含まれる場合、Perplexity に「どのツールを呼び出すか」をメタプロンプトで問い合わせ、OpenAI 互換の `tool_calls` を返却します
- **モデルリスト更新** – Claude 4.6、GPT 5.4、Gemini 3.1 など 22 の検証済みモデルをマッピング。`-search` サフィックスで Web 検索モードにも対応
- **プロバイダープレフィックス除去** – `pplx2api/claude-4.6-sonnet` 形式のモデル名から自動的にプレフィックスを除去します

---

## Features
