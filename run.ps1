param([switch]$Stop)

$app = "pplx2api"
$dir = Split-Path -Parent $PSCommandPath

if ($Stop) {
    Get-Process -Name $app -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host "Service stopped"
    return
}

# Stop old process
Get-Process -Name $app -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep 1

# Read .env
$envContent = Get-Content "$dir\.env" -Raw
$sessions = ($envContent -split "`n" | Where-Object { $_ -match '^SESSIONS=' }) -replace '^SESSIONS=', '' -replace "`r",''
$apikey = ($envContent -split "`n" | Where-Object { $_ -match '^APIKEY=' }) -replace '^APIKEY=', '' -replace "`r",''

# Set env for child process
$env:SESSIONS = $sessions
$env:APIKEY = $apikey
$env:ADDRESS = "0.0.0.0:18080"
$env:IS_INCOGNITO = "true"
$env:IGNORE_MODEL_MONITORING = "true"
$env:NO_ROLE_PREFIX = "true"
$env:MAX_CHAT_HISTORY_LENGTH = "1000000"

# Start process
$p = Start-Process -FilePath "$dir\$app.exe" -WorkingDirectory $dir -WindowStyle Hidden -PassThru

Start-Sleep 2

if (-not $p.HasExited) {
    Write-Host "========================================"
    Write-Host "  $app started (PID: $($p.Id))"
    Write-Host "========================================"
    Write-Host "  URL: http://localhost:18080"
    Write-Host "  Key: $apikey"
    Write-Host "========================================"
    try {
        $r = Invoke-RestMethod http://localhost:18080/v1/models -Headers @{"Authorization"="Bearer $apikey"} -ErrorAction Stop
        Write-Host "  Status: OK ($($r.data.count) models)"
    } catch {
        Write-Host "  Status: Starting..."
    }
} else {
    Write-Host "Failed to start!"
}
