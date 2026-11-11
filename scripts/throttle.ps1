# 采集节流状态（被风控后自动降速/暂停，成功后逐步恢复）
# 用法： . .\throttle.ps1  然后调用 Get-ThrottleDelay / Register-Block / Register-Success

$THROTTLE_FILE = 'E:\ai\data\scrape_throttle.json'

function Get-ThrottleState {
  $state = @{ level = 0; pausedUntil = ''; lastBlockAt = ''; consecutiveOk = 0 }
  if (Test-Path $THROTTLE_FILE) {
    try { $state = Get-Content $THROTTLE_FILE -Raw -Encoding UTF8 | ConvertFrom-Json } catch {}
  }
  return $state
}

function Save-ThrottleState($state) {
  $state | ConvertTo-Json -Depth 4 | Set-Content $THROTTLE_FILE -Encoding UTF8
}

# 返回剩余暂停秒数（0 表示可以跑）
function Get-ThrottlePause {
  $state = Get-ThrottleState
  if (-not $state.pausedUntil) { return 0 }
  try {
    $until = [datetime]::Parse($state.pausedUntil)
    $left = ($until - (Get-Date)).TotalSeconds
    if ($left -gt 0) { return [int]$left }
  } catch {}
  return 0
}

# 关键词之间 / 笔记之间的等待秒数
function Get-ThrottleDelay([int]$min, [int]$max) {
  $state = Get-ThrottleState
  $factor = @{ 0 = 1; 1 = 3; 2 = 6; 3 = 10 }[[int]$state.level]
  if (-not $factor) { $factor = 1 }
  return Get-Random -Minimum ([int]($min * $factor)) -Maximum ([int]($max * $factor) + 1)
}

# 被风控：升级等级并暂停
function Register-Block([string]$reason) {
  $state = Get-ThrottleState
  $level = [Math]::Min(3, [int]$state.level + 1)

  # 暂停时长：L1 10 分钟 / L2 40 分钟 / L3 2 小时
  $pauseMinutes = @{ 1 = 10; 2 = 40; 3 = 120 }[$level]
  if (-not $pauseMinutes) { $pauseMinutes = 10 }

  $state = @{
    level         = $level
    pausedUntil   = (Get-Date).AddMinutes($pauseMinutes).ToString('yyyy-MM-dd HH:mm:ss')
    lastBlockAt   = (Get-Date).ToString('yyyy-MM-dd HH:mm:ss')
    consecutiveOk = 0
    reason        = $reason
  }

  Save-ThrottleState $state
  return $state
}

# 顺利跑完：降一级（不低于 0）
function Register-Success {
  $state = Get-ThrottleState
  $level = [Math]::Max(0, [int]$state.level - 1)

  $state = @{
    level         = $level
    pausedUntil   = ''
    lastBlockAt   = $state.lastBlockAt
    consecutiveOk = [int]$state.consecutiveOk + 1
    reason        = ''
  }

  Save-ThrottleState $state
  return $state
}

function Write-ThrottleStatus {
  $state = Get-ThrottleState
  $pause = Get-ThrottlePause
  if ($pause -gt 0) {
    Write-Host ("[节流] 等级 L{0}，暂停中，还剩 {1} 分钟" -f $state.level, [int]($pause / 60))
  } elseif ($state.level -gt 0) {
    Write-Host ("[节流] 等级 L{0}（等待间隔已放大）" -f $state.level)
  }
}
