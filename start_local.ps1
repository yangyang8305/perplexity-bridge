# 停止旧进程
Get-Process -Name pplx2api -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 2

# 设置环境变量
$env:SESSIONS = (Get-Content "$PSScriptRoot\.env" | Where-Object { $_ -match '^SESSIONS=' } | ForEach-Object { $_ -replace '^SESSIONS=', '' })
$env:APIKEY = (Get-Content "$PSScriptRoot\.env" | Where-Object { $_ -match '^APIKEY=' } | ForEach-Object { $_ -replace '^APIKEY=', '' })
$env:ADDRESS = "0.0.0.0:18080"
$env:IS_INCOGNITO = "true"
$env:IGNORE_MODEL_MONITORING = "true"
$env:NO_ROLE_PREFIX = "true"
$env:MAX_CHAT_HISTORY_LENGTH = "1000000"

# 启动服务
$p = Start-Process -FilePath "$PSScriptRoot\pplx2api.exe" -WorkingDirectory $PSScriptRoot -NoNewWindow -PassThru

Start-Sleep -Seconds 2

if (-not $p.HasExited) {
    Write-Host "========================================"
    Write-Host "  Pplx2Api 已启动 (PID: $($p.Id))"
    Write-Host "========================================"
    Write-Host "  访问地址: http://localhost:18080"
    Write-Host "  API Key:  $env:APIKEY"
    Write-Host "========================================"
    
    # 测试连接
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:18080/health" -ErrorAction Stop
        Write-Host "  健康检查: OK"
    } catch {
        Write-Host "  健康检查: 等待中..."
    }
} else {
    Write-Host "启动失败！"
}
