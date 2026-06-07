# perplexity-bridge

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
