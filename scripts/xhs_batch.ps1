param(
  [string]$KeywordsCsv = '鸡蛋批发',
  [int]$NotesPerKeyword = 3,
  [int]$MaxDays = 180,
  [int]$Limit = 30,
  [string]$Exclude = '',
  [int]$CdpPort = 9333,
  [switch]$SkipPush
)

$ErrorActionPreference = 'Continue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

if ($CdpPort -gt 0) { $env:CDP = "http://127.0.0.1:$CdpPort" }

New-Item -ItemType Directory -Force -Path 'E:\ai\data\tmp' | Out-Null
$env:TEMP = 'E:\ai\data\tmp'
$env:TMP = 'E:\ai\data\tmp'

Set-Location 'E:\ai\scraper'

. 'E:\ai\scripts\scrape_seen.ps1'
# 采集节流：被风控会暂停并降速
. 'E:\ai\scripts\throttle.ps1'

$pauseLeft = Get-ThrottlePause

if ($pauseLeft -gt 0) {
  Write-Host ("[节流] 风控暂停中，还需 {0} 分钟，本次跳过" -f [int]($pauseLeft / 60))
  exit 0
}

Write-ThrottleStatus

$keywords = $KeywordsCsv -split '[,;]' | ForEach-Object { $_.Trim() } | Where-Object { $_ }
$excludeArg = @()
if ($Exclude) { $excludeArg = @('--exclude', $Exclude) }

$script:dupSkipped = 0
$totalNotes = 0
$totalComments = 0

Write-Host "开始采集小红书: 关键词=$($keywords -join '/')"

foreach ($kw in $keywords) {
  $env:XHS_KEYWORD = $kw
  Write-Host "[$kw] 搜索笔记..."
  $collectOut = node xhs_collect.mjs $kw --limit $NotesPerKeyword 2> E:\ai\data\logs\xhs_node.err

  if ($LASTEXITCODE -eq 3) {
    $st = Register-Block 'captcha'
    Write-Host ("BLOCKED captcha → 降速到 L{0}，暂停 {1} 分钟" -f $st.level, 10 * $st.level)
    exit 3
  }

  if ($LASTEXITCODE -ne 0) {
    Write-Host "[$kw] 搜索失败(exit=$LASTEXITCODE)，10 秒后重试一次"
    Start-Sleep -Seconds 10
    $collectOut = node xhs_collect.mjs $kw --limit $NotesPerKeyword 2> E:\ai\data\logs\xhs_node.err

    if ($LASTEXITCODE -ne 0) {
      Write-Host "[$kw] 重试仍失败，跳过该关键词"
      continue
    }
  }

  $file = "E:\ai\data\xhs_$($kw -replace '[\\/:*?"<>|]', '_').json"

  if (-not (Test-Path $file)) {
    Write-Host "[$kw] 没有结果文件，跳过"
    continue
  }

  $data = Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json

  if ([int]$data.total -eq 0) {
    Write-Host "[$kw] 搜索结果为 0，8 秒后重试"
    Start-Sleep -Seconds 8
    node xhs_collect.mjs $kw --limit $NotesPerKeyword 2> E:\ai\data\logs\xhs_node.err | Out-Null

    if (Test-Path $file) { $data = Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json }
  }

  # 只保留和关键词相关的笔记（小红书搜索会混进大量推荐内容）
  $tokens = $kw -split '[,;]' | ForEach-Object { $_.Trim() } | Where-Object { $_ }
  $relevant = @()

  foreach ($it in $data.items) {
    $text = ("" + $it.title + " " + $it.text).Trim()

    # 卡片没抓到标题/正文时无法判断，放行交给评论层过滤
    if ($text.Length -lt 4) { $relevant += $it; continue }

    $hit = $false

    foreach ($tk in $tokens) {
      if ($tk -and $text.Contains($tk)) { $hit = $true; break }
      # 关键词较长时允许拆词命中（如「鸡蛋批发」命中「鸡蛋」）
      if ($tk.Length -ge 4 -and $text.Contains($tk.Substring(0, 2))) { $hit = $true; break }
    }

    if ($hit) { $relevant += $it }
  }

  Write-Host "[$kw] 共 $($data.total) 条笔记，相关性过滤后 $($relevant.Count) 条（原 $($data.items.Count) 条）"
  $data.items = $relevant

  if ($relevant.Count -eq 0) {
    Write-Host "[$kw] 没有相关笔记，跳过"
    continue
  }

  $seen = Load-Seen 'xhs'
  Clear-OldSeen 'xhs' $seen | Out-Null

  foreach ($item in $data.items) {
    if (Test-SeenRecently 'xhs' $item.id $seen) {
      Write-Host "  笔记 $($item.id) 7 天内抓过，跳过"
      $script:dupSkipped++
      continue
    }

    Write-Host "  笔记 $($item.id) $($item.title)"
    node xhs_comments.mjs $item.id --days $MaxDays --limit $Limit --href "$($item.href)" @excludeArg 2>&1 | Out-Null

    if ($LASTEXITCODE -eq 3) {
      Write-Host "BLOCKED captcha on note $($item.id)"
      exit 3
    }

    Set-Seen 'xhs' $item.id $seen
    Save-Seen 'xhs' $seen

    $cf = "E:\ai\data\xhs_comments_$($item.id).json"

    if (-not (Test-Path $cf)) {
      Write-Host "    评论读取失败，5 秒后重试"
      Start-Sleep -Seconds 5
      node xhs_comments.mjs $item.id --days $MaxDays --limit $Limit --href "$($item.href)" @excludeArg 2>&1 | Out-Null
    }

    if (Test-Path $cf) {
      try {
        $c = Get-Content $cf -Raw -Encoding UTF8 | ConvertFrom-Json
        $totalComments += [int]$c.stats.kept
        $totalNotes += 1
        Write-Host "    评论 $($c.stats.raw) 条，保留 $($c.stats.kept) 条"
      } catch {}
    }

    Start-Sleep -Seconds (Get-ThrottleDelay 6 12)
  }

  Start-Sleep -Seconds (Get-ThrottleDelay 15 30)
}

if (-not $SkipPush) {
  Write-Host "推送到后台..."
  $pushOut = node xhs_push.mjs 'E:\ai\data' 2>&1
  Write-Host "  $($pushOut | Select-Object -Last 1)"
}

Register-Success | Out-Null
Write-Host "采集完成"
Write-Host ("STATS " + (@{ videos = $totalNotes; comments = $totalComments; leads = 0; skippedDup = [int]$script:dupSkipped } | ConvertTo-Json -Compress))
