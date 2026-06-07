# pplx2api Debug 总结

## 问题现象
所有模型返回的内容都是 `Display Model: turbo` 或 `Display Model: gpt5_nano`，没有真实回答；或者中文返回乱码。

## 故障原因（三个问题叠加）

### 问题 1：SESSIONS cookie 过期
- `pplx2api` 通过 Perplexity 的 `__Secure-next-auth.session-token` cookie 调用其内部 API
- 该 cookie 有效期短，过期后 Perplexity 不会拒绝请求（仍返回 HTTP 200），而是返回降级/占位内容
- 现象：返回 `Display Model: turbo`（旧过期 cookie）

### 问题 2：模型监控默认开启
- `pplx2api` 默认开启了模型监控（`IGNORE_MODEL_MONITORING` 默认 `false`）
- 当 Perplexity 实际使用的模型与你请求的不一致时，会在回答末尾追加一行：
  `\n\n---\nDisplay Model: xxx`
- 这行附加文本让用户误认为**只有**这个信息

### 问题 3：req 库的 ImpersonateChrome() 导致中文乱码（核心 Bug）
- 源码中 `req.C().ImpersonateChrome()` 用于模拟 Chrome 浏览器的 TLS 指纹
- 但这个函数会导致从 Perplexity SSE 流读取的中文 UTF-8 字符损坏
- 移除 `ImpersonateChrome()` 后中文恢复正常
- 修复位置：`core/api.go` 中的 `NewClient` 函数

## 解决方案

### 更新 SESSIONS cookie
```
1. 浏览器打开 https://www.perplexity.ai/ 并登录
2. F12 → Application → Cookies → https://www.perplexity.ai
3. 找到 __Secure-next-auth.session-token，复制 Value
4. 编辑 C:\AI\Justtalktest\pplx2api_source\.env
5. 把 SESSIONS= 后面的内容替换成新复制的 cookie 值
6. 重启服务：运行 run.ps1 或 start.bat
```

### 屏蔽模型监控
在 `pplx2api_source\.env` 或环境变量中添加：
```
IGNORE_MODEL_MONITORING=true
```

### 中文乱码修复
已修改 `core/api.go` 中的 `NewClient` 函数，移除了 `ImpersonateChrome()` 调用。

## 验证方法
```powershell
# 测试英文
curl.exe -s -X POST http://localhost:18080/v1/chat/completions ^
  -H "Authorization: Bearer pplx2api2026" ^
  -H "Content-Type: application/json" ^
  -d "{\"model\":\"claude-4-6-sonnet\",\"messages\":[{\"role\":\"user\",\"content\":\"Say hello\"}],\"stream\":false}"

# 测试中文
curl.exe -s -X POST http://localhost:18080/v1/chat/completions ^
  -H "Authorization: Bearer pplx2api2026" ^
  -H "Content-Type: application/json" ^
  -d "{\"model\":\"claude-4-6-sonnet\",\"messages\":[{\"role\":\"user\",\"content\":\"你好\"}],\"stream\":false}"
```

## 当前配置
- **运行方式**: 本地编译运行（非 Docker）
- **源码位置**: `C:\AI\Justtalktest\pplx2api_source`
- **可执行文件**: `C:\AI\Justtalktest\pplx2api_source\pplx2api.exe`
- **配置文件**: `C:\AI\Justtalktest\pplx2api_source\.env`
- **启动脚本**: `C:\AI\Justtalktest\pplx2api_source\run.ps1`
- **访问地址**: http://localhost:18080
- **API Key**: pplx2api2026
- **支持模型**: claude-4-6-sonnet, gpt-5.4, gemini-3.1-pro (+ -search, -think 版本)
- **模型监控**: 已关闭 (IGNORE_MODEL_MONITORING=true)
- **角色前缀**: 已关闭 (NO_ROLE_PREFIX=true)

## 启动/停止命令
```powershell
# 启动
cd C:\AI\Justtalktest\pplx2api_source
.\run.ps1

# 停止
.\run.ps1 -Stop

# 或者用批处理
C:\AI\Justtalktest\pplx2api\start.bat
C:\AI\Justtalktest\pplx2api\stop.bat
```

## 如何重新编译
修改源码后，在 `pplx2api_source` 目录下运行：
```powershell
& "C:\Program Files\Go\bin\go.exe" build -o pplx2api.exe .
```

## 注意事项
- SESSIONS cookie 会定期过期，需要定期更新
- 修改代码后需要重新编译才能生效
- 源码在 `pplx2api_source` 目录，可以直接编辑 `.go` 文件修改逻辑
