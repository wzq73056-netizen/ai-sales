# 健康检查（由计划任务 DSH-AI-Sales-Health 每 2 分钟调用）
#
# 检查项：
#   1. 微信监听：进程在 + 心跳在 3 分钟内更新 + 有没有掉登录标记
#   2. 抖音私信监听：进程在 + 心跳
#   3. 小红书私信监听：进程在 + 心跳
#   4. 语音转写服务 8898
#   5. 后端 8888 / 前端 9527
#   6. CDP 9333 通不通（不通就重启浏览器）
#   7. 浏览器标签页卫生（脚本跑完会残留标签，顺手清掉）
#
# 有动作时调用 start-all.ps1（幂等，已在运行的会跳过）

param(
  [switch]$Verbose
)

$ErrorActionPreference = 'Continue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$NODE = 'D:\Program Files\nodejs\node.exe'
$START_ALL = 'E:\ai\scripts\start-all.ps1'
$LOGDIR = 'E:\ai\data\logs'
$LOG = Join-Path $LOGDIR 'health.log'

$actions = 0

function Write-Log($msg) {
  $line = '{0} {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg
  Write-Host $line

  if (-not (Test-Path $LOGDIR)) { New-Item -ItemType Directory -Path $LOGDIR -Force | Out-Null }

  Add-Content -Path $LOG -Value $line -Encoding utf8
}

function Test-Port([int]$port) {
  try {
    $client = New-Object System.Net.Sockets.TcpClient
    $client.Connect('127.0.0.1', $port)
    $client.Close()

    return $true
  } catch {
    return $false
  }
}

function Get-WatcherProcess([string]$pattern) {
  return Get-CimInstance Win32_Process -Filter "Name='node.exe' OR Name='python.exe'" |
    Where-Object { $_.CommandLine -like "*$pattern*" } |
    Select-Object -First 1
}

function Test-Heartbeat([string]$file, [double]$minutes) {
  $item = Get-Item $file -ErrorAction SilentlyContinue
  if (-not $item) { return $false }

  return ((Get-Date) - $item.LastWriteTime).TotalMinutes -le $minutes
}

# ---- 1. 微信监听 ----
$wxProc = Get-WatcherProcess 'wx_watch.py'
$wxBeat = Test-Heartbeat 'E:\ai\data\wx_heartbeat.txt' 3
$wxFlag = Get-Item 'E:\ai\data\wx_key_failed.flag' -ErrorAction SilentlyContinue

if ($wxFlag) {
  Write-Log '微信掉登录了（wx_key_failed.flag），需要扫码重新登录'
  $actions++

  # 主动推到手机（同一条 30 分钟内只推一次）
  & powershell -NoProfile -ExecutionPolicy Bypass -File 'E:\ai\scripts\notify.ps1' `
    -Title '微信掉线了' `
    -Body '需要重新扫码登录微信。扫码前不会自动回复任何消息。' 2>&1 | Out-Null
}

if (-not $wxProc) {
  Write-Log '微信监听未运行 → 拉起'
  $actions++
} elseif (-not $wxBeat) {
  Write-Log '微信监听心跳超过 3 分钟（假活）→ 重启'

  & powershell -NoProfile -ExecutionPolicy Bypass -File 'E:\ai\scripts\notify.ps1' `
    -Title '微信监听假活' `
    -Body '心跳超过 3 分钟没更新，已自动重启监听进程。' 2>&1 | Out-Null
  $wxProc | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
  $actions++
}

# ---- 2. 抖音私信监听 ----
$dyHearts = Get-ChildItem 'E:\ai\data' -Filter 'dy_heartbeat-*.txt' -ErrorAction SilentlyContinue
$dyAlive = $false

foreach ($file in $dyHearts) {
  if (((Get-Date) - $file.LastWriteTime).TotalMinutes -le 3) { $dyAlive = $true }
}

$dyProc = Get-WatcherProcess 'dy_dm_watch.mjs'

if (-not $dyProc) {
  Write-Log '抖音私信监听未运行 → 拉起'
  $actions++
} elseif (-not $dyAlive) {
  Write-Log '抖音私信监听心跳超时 → 重启'
  $dyProc | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
  $actions++
}

# ---- 3. 小红书私信监听 ----
$xhsProc = Get-WatcherProcess 'xhs_dm_watch.mjs'
$xhsBeat = Test-Heartbeat 'E:\ai\data\xhs_heartbeat.txt' 3

if (-not $xhsProc) {
  Write-Log '小红书私信监听未运行 → 拉起'
  $actions++
} elseif (-not $xhsBeat) {
  Write-Log '小红书私信监听心跳超时 → 重启'
  $xhsProc | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
  $actions++
}

# ---- 4. 语音转写服务 ----
if (-not (Test-Port 8898)) {
  Write-Log '语音转写服务未运行 → 拉起'
  $actions++
}

# ---- 5. 后端 / 前端 ----
if (-not (Get-Process gva -ErrorAction SilentlyContinue)) {
  Write-Log '后端未运行 → 拉起'
  $actions++
} elseif (-not (Test-Port 8888)) {
  Write-Log '后端进程在但端口不通 → 重启'
  Get-Process gva -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
  $actions++
}

if (-not (Test-Port 9527)) {
  Write-Log '前端未运行 → 拉起'
  $actions++
}

# ---- 6. 浏览器 CDP ----
if (-not (Test-Port 9333)) {
  Write-Log 'CDP 9333 不通 → 重启浏览器'
  $actions++
}

# ---- 7. 浏览器标签页卫生 ----
if (Test-Port 9333) {
  try {
    $tidy = & $NODE 'E:\ai\scraper\tabs_tidy.mjs' 2>$null
    if ($tidy -match '"closed":([1-9]\d*)') {
      Write-Log ('清理残留浏览器标签 {0} 个' -f $matches[1])
      $actions++
    }
  } catch {
    if ($Verbose) { Write-Log '标签清理失败，跳过' }
  }
}

# ---- 有动作才调 start-all（幂等） ----
if ($actions -gt 0) {
  if ($Verbose) { Write-Log "触发项 $actions 个 → 执行 start-all" }

  try {
    & powershell -NoProfile -ExecutionPolicy Bypass -File $START_ALL 2>&1 | Out-Null
    Write-Log "已执行启动脚本（触发项 $actions 个）"
  } catch {
    Write-Log ("启动脚本执行失败：{0}" -f $_.Exception.Message)
  }
} elseif ($Verbose) {
  Write-Log '一切正常，无动作'
}

exit 0
