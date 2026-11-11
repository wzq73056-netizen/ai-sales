# 一键启动所有服务（后端 + 前端 + 语音 + 微信监听 + 抖音账号）
# 用法： .\start-all.ps1
# 停止： .\start-all.ps1 -Stop
param(
  [switch]$Stop
)

$ErrorActionPreference = 'Continue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$PY = 'C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\python\python.exe'
$NODE = 'C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\node\bin\node.exe'
$EDGE = 'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
$BACKEND = 'E:\ai\gin-vue-admin\server\bin\gva.exe'
$BACKEND_DIR = 'E:\ai\gin-vue-admin\server'
$WATCHER = 'E:\ai\scripts\wx_watch.py'
$API_BASE = 'http://127.0.0.1:8888'
$LOGDIR = 'E:\ai\data\logs'
$FRONT_DIR = 'E:\ai\soybean-admin'

New-Item -ItemType Directory -Force -Path $LOGDIR | Out-Null

function Stop-All {
  Write-Host '停止服务…'

  Get-CimInstance Win32_Process -Filter "Name='python.exe'" |
    Where-Object { $_.CommandLine -like '*wx_watch*' } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

  Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
    Where-Object { $_.CommandLine -like '*dy_dm_watch*' -or $_.CommandLine -like '*xhs_dm_watch*' } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

  Get-CimInstance Win32_Process -Filter "Name='msedge.exe'" |
    Where-Object { $_.CommandLine -like '*edge-profile*' } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

  Get-Process gva -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

  Get-NetTCPConnection -LocalPort 9527 -State Listen -ErrorAction SilentlyContinue |
    ForEach-Object { Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue }

  Write-Host '已停止'
}

function Start-Backend {
  if (Get-Process gva -ErrorAction SilentlyContinue) {
    Write-Host '后端已在运行'
    return
  }

  # 大模型配置从本机 E:\ai\data\llm.json 读取，不要把 key 写进脚本
  $llmFile = 'E:\ai\data\llm.json'
  if (Test-Path $llmFile) {
    try {
      $llm = Get-Content $llmFile -Raw -Encoding UTF8 | ConvertFrom-Json
      if ($llm.base_url) { $env:RAG_CHAT_BASE_URL = [string]$llm.base_url }
      if ($llm.api_key)  { $env:RAG_CHAT_API_KEY  = [string]$llm.api_key }
      if ($llm.model)    { $env:RAG_CHAT_MODEL    = [string]$llm.model }
    } catch {
      Write-Host "llm.json 解析失败：$($_.Exception.Message)"
    }
  } else {
    Write-Host "未找到 $llmFile，大模型功能不可用"
  }

  $env:DM_KB_ID = '2002'

  Start-Process -FilePath $BACKEND -WindowStyle Hidden -WorkingDirectory $BACKEND_DIR -RedirectStandardOutput "$LOGDIR\backend.out.log" -RedirectStandardError "$LOGDIR\backend.err.log"

  for ($i = 0; $i -lt 40; $i++) {
    Start-Sleep -Milliseconds 500
    if ((Test-NetConnection -ComputerName 127.0.0.1 -Port 8888 -WarningAction SilentlyContinue).TcpTestSucceeded) {
      Write-Host '后端已就绪 (8888)'
      return
    }
  }

  Write-Host '后端启动超时'
}

function Start-Frontend {
  # 前端已由 gva.exe 托管（dist 打进后端），不再需要 dev server
  if ((Test-NetConnection -ComputerName 127.0.0.1 -Port 8888 -WarningAction SilentlyContinue).TcpTestSucceeded) {
    Write-Host "前端随后端提供: http://127.0.0.1:8888/"
  } else {
    Write-Host "后端未就绪，前端页面无法访问"
  }
}


function Start-Audio {
  if ((Test-NetConnection -ComputerName 127.0.0.1 -Port 8898 -WarningAction SilentlyContinue).TcpTestSucceeded) {
    Write-Host '语音转写服务已在运行'
    return
  }

  $py311 = 'C:\Users\11\AppData\Local\Programs\Python\Python311\python.exe'
  if (-not (Test-Path $py311)) {
    Write-Host '语音服务跳过：找不到 Python 3.11'
    return
  }

  Start-Process -FilePath $py311 -ArgumentList 'serve_audio.py', '--port', '8898' -WindowStyle Hidden -WorkingDirectory 'E:\ai\ai-auto\ruoyi-ai\scripts' -RedirectStandardOutput "$LOGDIR\audio.out.log" -RedirectStandardError "$LOGDIR\audio.err.log"

  for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Milliseconds 500
    if ((Test-NetConnection -ComputerName 127.0.0.1 -Port 8898 -WarningAction SilentlyContinue).TcpTestSucceeded) {
      Write-Host '语音转写服务已就绪 (8898)'
      return
    }
  }

  Write-Host '语音转写服务启动超时'
}

function Start-Watcher {
  $running = Get-CimInstance Win32_Process -Filter "Name='python.exe'" |
    Where-Object { $_.CommandLine -like '*wx_watch*' }

  if ($running) {
    Write-Host '微信监听已在运行'
    return
  }

  Start-Process -FilePath $PY -ArgumentList $WATCHER, '--interval', '1.5', '--reply', '--send' -WindowStyle Hidden -WorkingDirectory 'E:\ai\scripts' -RedirectStandardOutput "$LOGDIR\wxwatch.out.log" -RedirectStandardError "$LOGDIR\wxwatch.err.log"

  Write-Host '微信监听已启动（自动回复 + 转人工 + 客户打分）'
}

function Get-ApiHeaders {
  $login = Invoke-RestMethod -Uri "$API_BASE/base/login" -Method Post -Body (@{ username = 'admin'; password = '123456'; captcha = ''; captchaId = '' } | ConvertTo-Json) -ContentType 'application/json' -TimeoutSec 10

  return @{ 'x-token' = $login.data.token }
}

function Start-DouyinWatch {
  $devices = @()

  try {
    $headers = Get-ApiHeaders
    $devices = (Invoke-RestMethod -Uri "$API_BASE/lead/getDouyinDevices" -Headers $headers -TimeoutSec 10).data.list
  } catch {
    Write-Host '拿不到抖音账号列表，跳过抖音'
    return
  }

  # 只处理抖音账号，小红书账号另有监听
  $enabled = @($devices | Where-Object { $_.Status -eq 1 -and ($_.Platform -eq $null -or $_.Platform -eq "" -or $_.Platform -eq "douyin") })

  if ($enabled.Count -eq 0) {
    Write-Host '没有启用的抖音账号'
    return
  }

  foreach ($device in $enabled) {
    $port = [int]$device.CdpPort
    $id = $device.Id

    if (-not (Test-NetConnection -ComputerName 127.0.0.1 -Port $port -WarningAction SilentlyContinue).TcpTestSucceeded) {
      # 每个账号可以用不同的浏览器
      $exe = if ($device.BrowserPath) { $device.BrowserPath } else { $EDGE }
      if (-not (Test-Path $exe)) {
        Write-Host "账号 #$id 的浏览器不存在：$exe"
        continue
      }

      Start-Process -FilePath $exe -ArgumentList "--remote-debugging-port=$port", "--user-data-dir=$($device.ProfileDir)", '--no-first-run', '--no-default-browser-check', 'https://www.douyin.com'

      for ($i = 0; $i -lt 40; $i++) {
        Start-Sleep -Milliseconds 500
        if ((Test-NetConnection -ComputerName 127.0.0.1 -Port $port -WarningAction SilentlyContinue).TcpTestSucceeded) { break }
      }

      $browserName = if ($device.BrowserName) { $device.BrowserName } else { "Edge" }
      Write-Host "抖音浏览器 #$id $($device.Name) 已启动（$browserName，端口 $port）"
    }

    $pattern = "*dy_dm_watch*--device $id*"
    $running = Get-CimInstance Win32_Process -Filter "Name='node.exe'" | Where-Object { $_.CommandLine -like $pattern }

    if ($running) {
      Write-Host "抖音私信监听 #$id 已在运行"
      continue
    }

    Start-Process -FilePath $NODE -ArgumentList 'E:\ai\scraper\dy_dm_watch.mjs', '--interval', '60', '--send', '--device', "$id" -WindowStyle Hidden -WorkingDirectory 'E:\ai\scraper' -RedirectStandardOutput "$LOGDIR\dydm-$id.out.log" -RedirectStandardError "$LOGDIR\dydm-$id.err.log"

    Write-Host "抖音私信监听 #$id $($device.Name) 已启动"
  }
}

function Start-XhsWatch {
  $running = Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
    Where-Object { $_.CommandLine -like '*xhs_dm_watch*' }

  if ($running) {
    Write-Host '小红书私信监听已在运行'
    return
  }

  Start-Process -FilePath $NODE -ArgumentList 'E:\ai\scraper\xhs_dm_watch.mjs', '--interval', '60', '--send' -WindowStyle Hidden -WorkingDirectory 'E:\ai\scraper' -RedirectStandardOutput "$LOGDIR\xhsdm.out.log" -RedirectStandardError "$LOGDIR\xhsdm.err.log"

  Write-Host '小红书私信监听已启动（白名单：冬冬瓜咚）'
}

if ($Stop) {
  Stop-All
  return
}

Write-Host "=== 启动 $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ==="
Start-Backend
Start-Frontend
Start-Audio
Start-Watcher
Start-DouyinWatch
Start-XhsWatch
Write-Host '=== 完成 ==='