# 采集去重：同一篇笔记/视频 7 天内不重复抓
$SEEN_DAYS = 7

function Get-SeenFile([string]$platform) { return "E:\ai\data\seen_$platform.json" }

function Load-Seen([string]$platform) {
  $file = Get-SeenFile $platform
  $map = @{}

  if (Test-Path $file) {
    try {
      $raw = Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json
      foreach ($p in $raw.PSObject.Properties) { $map[$p.Name] = [string]$p.Value }
    } catch {}
  }

  return $map
}

function Save-Seen([string]$platform, $map) {
  $map | ConvertTo-Json -Depth 3 | Set-Content (Get-SeenFile $platform) -Encoding UTF8
}

# 最近抓过就返回 true，可以跳过
function Test-SeenRecently([string]$platform, [string]$id, $map) {
  if (-not $map.ContainsKey($id)) { return $false }

  try {
    $last = [datetime]::Parse($map[$id])
    return ((Get-Date) - $last).TotalDays -lt $SEEN_DAYS
  } catch {
    return $false
  }
}

function Set-Seen([string]$platform, [string]$id, $map) {
  $map[$id] = (Get-Date).ToString('yyyy-MM-dd HH:mm:ss')
}

# 清理超过 30 天的旧记录，避免文件无限增长
function Clear-OldSeen([string]$platform, $map) {
  $cut = (Get-Date).AddDays(-30)
  $removed = 0

  foreach ($key in @($map.Keys)) {
    try {
      if ([datetime]::Parse($map[$key]) -lt $cut) {
        $map.Remove($key)
        $removed++
      }
    } catch {
      $map.Remove($key)
      $removed++
    }
  }

  return $removed
}
