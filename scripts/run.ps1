$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$FrontendDir = Join-Path $Root "frontend"
$LogDir = Join-Path $Root "logs"
$EnvFile = Join-Path $BackendDir ".env.local"
$BackendPort = 8091

function Stop-Port($Port) {
  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne 0) {
      Stop-Process -Id $connection.OwningProcess -Force -ErrorAction SilentlyContinue
    }
  }
}

function Load-Env($Path) {
  if (-not (Test-Path $Path)) {
    Write-Host "No backend/.env.local found. Defaults will be used." -ForegroundColor Yellow
    return
  }
  Get-Content -Encoding UTF8 $Path | ForEach-Object {
    if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
      [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2], "Process")
    }
  }
  if ([Environment]::GetEnvironmentVariable("BITAPI_HTTP_ADDR", "Process") -eq ":8080") {
    [Environment]::SetEnvironmentVariable("BITAPI_HTTP_ADDR", ":$BackendPort", "Process")
  }
}

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

Write-Host "Stopping old services..." -ForegroundColor Cyan
Stop-Port 8080
Stop-Port 5173
Stop-Port 5181
Stop-Port $BackendPort

Load-Env $EnvFile

if (-not (Test-Path (Join-Path $FrontendDir "node_modules"))) {
  Write-Host "Installing frontend dependencies..." -ForegroundColor Cyan
  Start-Process -FilePath "npm.cmd" -ArgumentList @("install") -WorkingDirectory $FrontendDir -Wait -WindowStyle Hidden
}

Write-Host "Building frontend dist..." -ForegroundColor Cyan
Start-Process -FilePath "npm.cmd" -ArgumentList @("run", "build") -WorkingDirectory $FrontendDir -Wait -WindowStyle Hidden

Write-Host "Starting backend http://localhost:$BackendPort ..." -ForegroundColor Cyan
$backendOut = Join-Path $LogDir "backend.log"
$backendErr = Join-Path $LogDir "backend.err.log"
$backend = Start-Process -FilePath "go" -ArgumentList @("run", ".\cmd\server") -WorkingDirectory $BackendDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $backendOut -RedirectStandardError $backendErr
$backend.Id | Set-Content -Path (Join-Path $BackendDir ".server.pid") -Encoding ASCII

function Test-Healthy($Url) {
  try {
    Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 2 | Out-Null
    return $true
  } catch {
    return $false
  }
}

$deadline = (Get-Date).AddSeconds(30)
while ((Get-Date) -lt $deadline) {
  if ($backend.HasExited) {
    Write-Host "Backend failed to start. Check $backendErr" -ForegroundColor Red
    Get-Content $backendErr -ErrorAction SilentlyContinue | Select-Object -Last 80 | Out-Host
    exit 1
  }
  if (Test-Healthy "http://127.0.0.1:$BackendPort/health") {
    break
  }
  Start-Sleep -Seconds 1
}
if (-not (Test-Healthy "http://127.0.0.1:$BackendPort/health")) {
  Write-Host "Backend did not become healthy. Check $backendErr" -ForegroundColor Red
  Get-Content $backendErr -ErrorAction SilentlyContinue | Select-Object -Last 80 | Out-Host
  exit 1
}
Write-Host "Started." -ForegroundColor Green
Write-Host "Web and API: http://localhost:$BackendPort"
Write-Host "Admin login: http://localhost:$BackendPort/auth/login"
Write-Host "Backend log: $backendOut"
