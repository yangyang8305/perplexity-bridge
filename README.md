# perplexity-bridge

> 将 Perplexity AI 包装成完整兼容 OpenAI API 的本地网关，支持流式输出、图片上传、Function Calling、多 Session 轮转与自动刷新。

---

## 功能特性

- **OpenAI API 兼容** — `/v1/chat/completions`、`/v1/models` 等标准接口，可直接接入任意支持 OpenAI 的客户端
- **流式输出** — Server-Sent Events (SSE) 实时流式返回
- **多模型支持** — Claude、GPT、Gemini、Grok、Kimi、Sonar 系列等；模型名加 `-search` 后缀开启网络搜索
- **Function Calling** — 两阶段工具选择，支持并行工具调用，输出格式完整符合 OpenAI spec
- **图片上传** — 支持 base64 图片自动上传到 Perplexity
- **长文本上传** — 超过限制长度时自动将上下文上传为文件
- **多 Session 轮转** — 支持配置多个 Session，自动轮转负载均衡，失败自动重试
- **Session 自动刷新** — 定时刷新 Cookie，将有效 Session 持久化到 `sessions.json`
- **Web 管理面板** — 内置 Dashboard 可查看运行状态、手动触发刷新、重载配置
- **思考链支持** — 自动识别 `<think>` 思考内容并包装输出
- **搜索结果嵌入** — 开启搜索模式时自动将网页引用附加在回复后

---

## 支持的模型

| 模型 ID | 说明 |
|---------|------|
| `claude-4.6-sonnet` | Claude 4.6 Sonnet |
| `claude-4.6-sonnet-think` | Claude 4.6 Sonnet 思考模式 |
| `claude-4.5-sonnet` | Claude 4.5 Sonnet |
| `claude-4.0-sonnet` | Claude 4.0 Sonnet |
| `gpt-5.4` | GPT-5.4 |
| `gpt-5.4-think` | GPT-5.4 思考模式 |
| `gpt-5.2` | GPT-5.2 |
| `gpt-4.1` | GPT-4.1 |
| `gpt-4o` | GPT-4o |
| `o4-mini` | o4-mini |
| `gemini-3.1-pro` | Gemini 3.1 Pro |
| `gemini-3-pro` | Gemini 3 Pro |
| `grok-4.1` | Grok 4.1 |
| `kimi` | Kimi |
| `kimi-k2` | Kimi K2 |
| `kimi-k2-think` | Kimi K2 思考模式 |
| `sonar` | Sonar |
| `sonar-pro` | Sonar Pro |
| `sonar-reasoning` | Sonar Reasoning |
| `sonar-reasoning-pro` | Sonar Reasoning Pro |
| `sonar-deep-research` | Sonar Deep Research |
| `deepseek-r1` | DeepSeek R1 |
| `claude-4.6-opus` ☆ | Claude 4.6 Opus（需 Max 订阅） |
| `claude-4.6-opus-think` ☆ | Claude 4.6 Opus 思考模式（需 Max 订阅） |

> ☆ Max 订阅专属模型需设置 `IS_MAX_SUBSCRIBE=true` 才会出现在 `/v1/models` 列表中。
>
> 所有模型 ID 后加 `-search` 尾缀可开启网络搜索，例如 `claude-4.6-sonnet-search`。

---

## 快速部署

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e SESSIONS="your_session_token_here" \
  -e APIKEY="your_api_key" \
  --name perplexity-bridge \
  ghcr.io/yangyang8305/perplexity-bridge:latest
```

### Docker Compose

```yaml
services:
  perplexity-bridge:
    image: ghcr.io/yangyang8305/perplexity-bridge:latest
    ports:
      - "8080:8080"
    environment:
      SESSIONS: "session_token_1,session_token_2"
      APIKEY: "your_api_key"
      IS_INCOGNITO: "true"
      IS_MAX_SUBSCRIBE: "false"
    restart: unless-stopped
```

### 本地运行

```bash
# 复制配置文件
cp .env.example .env
# 编辑 .env 填入你的 Session
vim .env
# 运行
go run main.go
```

Windows 可直接双击 `启动服务.bat` 或运行 `start_local.ps1`。

---

## 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `SESSIONS` | 是 | — | Session Token，多个用逗号分隔 |
| `APIKEY` | 否 | 空 | 访问此服务的 API Key（空则不鉴权） |
| `ADDRESS` | 否 | `:8080` | 监听地址 |
| `IS_INCOGNITO` | 否 | `true` | 是否开启隐身模式 |
| `IS_MAX_SUBSCRIBE` | 否 | `false` | 是否是 Max 订阅（解锁 Opus 等模型） |
| `PROXY` | 否 | 空 | HTTP/SOCKS5 代理地址 |
| `MAX_CHAT_HISTORY_LENGTH` | 否 | `10000` | 超过此长度自动上传文本 |
| `PROMPT_FOR_FILE` | 否 | 内置 | 上传文件后的提示语 |
| `NO_ROLE_PREFIX` | 否 | `false` | 是否去掉角色前缀（System:/Human:/Assistant:） |
| `IGNORE_SEARCH_RESULT` | 否 | `false` | 是否隐藏搜索结果引用 |
| `SEARCH_RESULT_COMPATIBLE` | 否 | `false` | 搜索结果改用简洁格式 |
| `IGNORE_MODEL_MONITORING` | 否 | `false` | 关闭模型漂移警告日志 |
| `TIMEZONE` | 否 | `America/New_York` | 时区（如 `Asia/Shanghai`） |

---

## 接口说明

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/completions` | 聊天接口（兼容 OpenAI） |
| GET | `/v1/models` | 获取模型列表 |
| GET | `/health` | 健康检查 |
| GET | `/dashboard` | Web 管理面板 |
| GET | `/admin/status` | 运行状态 JSON |
| POST | `/admin/refresh` | 手动触发 Session 刷新 |
| POST | `/admin/reload` | 重载 `.env` 配置 |

---

## 获取 Session Token

1. 登录 [perplexity.ai](https://www.perplexity.ai)
2. 打开浏览器开发者工具 → Application → Cookies
3. 找到 `__Secure-next-auth.session-token`，复制其值
4. 填入 `SESSIONS` 环境变量

多个 Token 用逗号分隔：`SESSIONS=token1,token2,token3`

---

## 局限性

- Perplexity 本身不支持原生 Function Calling，本项目通过两次独立请求模拟工具调用（先选工具再填参数），效果依赖模型语义理解能力，不保证 100% 准确率
- Session Token 会定期过期，建议开启自动刷新
- 受 Perplexity 速率限制，高并发场景建议配置多个 Session

---

## License

MIT
