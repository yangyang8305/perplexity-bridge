# Changelog

## [Unreleased]

### 修复 (Bug Fixes)

#### 高优先级 🔴

- **`service/handle.go`** — 修复 retry 循环中当 Sessions 被并发清空为 0 时 `% 0` 导致程序 panic 的问题。现在每次循环开头先在 `RLock` 内快照当前 Session 数量，为 0 则直接跳出循环。同时将循环内所有 Config 字段读取集中到一次 `RLock` 快照块。
- **`core/tools.go`** — 修复 JSON 提取逻辑使用手写括号计数时，当工具参数值内包含 `{` 或 `}` 字符会错误截断 JSON 的问题。改用 `json.NewDecoder` 正确解析，不再被字符串内容干扰。
- **`job/cookie.go`** — 修复 `loadSessionsFromFile()` 替换 Sessions 后漏调 `Sr.ResetIndex()`，可能导致轮转索引越界。现在在 `RwMutex.Unlock()` 后立即调用。

#### 中优先级 🟡

- **`config/reload.go`** — 修复 `Reload()` 后漏调 `Sr.ResetIndex()` 和 `buildResponseModels()`，导致重载后轮转索引可能越界且 `/v1/models` 返回内容不更新。
- **`service/dashboard.go`** — 修复 `AdminRefreshHandler` 无防重入保护，短时间内多次触发会并发运行多个 `updateAllSessions()`。改用 `atomic.CompareAndSwapInt32` 实现防重入，已有刷新进行时返回 429。
- **`utils/role.go`** — 修复 `GetRolePrefix()` 读取 `NoRolePrefix` 时无 `RLock` 保护，与 `Reload()` 并发时存在数据竞态。
- **`utils/searchShow.go`** — 修复 `SearchShow()` 读取 `SearchResultCompatible` 时无 `RLock` 保护，同上。
- **`core/api.go`** — 修复 `timezone()` 读取 `Timezone` 字段无锁；修复 `HandleResponse()` 在流式循环中反复裸读 `IgnoreSerchResult`、`IgnoreModelMonitoring`。均改为循环前一次 `RLock` 快照。

#### 低优先级 🟢

- **`model/openai.go`** — 删除 3 个从未被引用的冗余 struct：`ChatCompletionRequest`、`ToolCallInfo`、`ToolCallFunctionInfo`。
- **`model/openai.go`** — 修复 stream 模式下 `tool_calls` 输出格式不符合 OpenAI spec 的问题。原来将所有 tool call 一次性放入单个 chunk，现改为每个 tool call 单独发一个 chunk，完整按照 OpenAI streaming 工具调用规范输出。

---

## 历史版本

详见 [Git 提交记录](https://github.com/yangyang8305/perplexity-bridge/commits/main)。
