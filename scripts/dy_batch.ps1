param(
  [string]$KeywordsCsv = '代理记账,注册公司,小规模纳税人,个体户记账',
  [int]$VideosPerKeyword = 3,
  [int]$MaxDays = 180,
  [int]$Limit = 10,
  [string]$City = '',
  [string]$Exclude = '',
  [string]$CrawlType = 'video',
  [int]$CdpPort = 9333,
  [switch]$SkipPush
)

$ErrorActionPreference = 'Continue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

# 指定用哪个抖音账号的浏览器
if ($CdpPort -gt 0) {
  $env:CDP = "http://127.0.0.1:$CdpPort"
}

New-Item -ItemType Directory -Force -Path 'E:\ai\data\tmp' | Out-Null
$env:TEMP = 'E:\ai\data\tmp'
$env:TMP = 'E:\ai\data\tmp'

. 'E:\ai\scripts\scrape_seen.ps1'

# 采集节流：被风控会暂停并降速
. 'E:\ai\scripts\throttle.ps1'

$pauseLeft = Get-ThrottlePause

if ($pauseLeft -gt 0) {
  Write-Host ("[节流] 风控暂停中，还需 {0} 分钟，本次跳过" -f [int]($pauseLeft / 60))
  exit 0
}

Write-ThrottleStatus

$scraper = 'E:\ai\scraper'
Set-Location $scraper

$keywords = $KeywordsCsv -split '[,;]' | ForEach-Object { $_.Trim() } | Where-Object { $_ }
$cityArg = @()
if ($City) { $cityArg = @('--city', $City) }

$excludeArg = @()
if ($Exclude) { $excludeArg = @('--exclude', $Exclude) }

$leadsFile = 'E:\ai\data\leads.json'
$beforeLeads = 0
if (Test-Path $leadsFile) {
  try { $beforeLeads = (Get-Content $leadsFile -Raw -Encoding UTF8 | ConvertFrom-Json).total } catch { $beforeLeads = 0 }
}

$totalVideos = 0
$totalAccounts = 0
$script:dupSkipped = 0
$totalComments = 0

Write-Host "开始采集: 关键词=$($keywords -join '/') 视频数=$VideosPerKeyword 天数=$MaxDays"

foreach ($kw in $keywords) {
  if ($CrawlType -eq 'account') {
    Write-Host "[$kw] 搜索用户（账号获客）..."
    $userOut = node dy_users.mjs $kw --limit $Limit 2>&1

    if ($LASTEXITCODE -ne 0) {
      ($userOut | Out-String) -split "`n" | Select-Object -Last 4 | ForEach-Object { Write-Host "  $_" }
    }

    $uf = "E:\ai\data\dy_users_$kw.json"

    if (Test-Path $uf) {
      try {
        $u = Get-Content $uf -Raw -Encoding UTF8 | ConvertFrom-Json
        $totalAccounts += [int]$u.found
        $totalVideos += [int]$u.kept
      } catch {}
    }

    Start-Sleep -Seconds (Get-ThrottleDelay 20 40)
    continue
  }

  Write-Host "[$kw] 搜索视频..."
  $collectOut = node dy_collect.mjs $kw 2>&1

  if ($LASTEXITCODE -eq 3) {
    $reason = (($collectOut | Out-String) -split "`n" | Select-String -Pattern '"reason"' | Select-Object -First 1)
    Write-Host "BLOCKED $reason"
    $st = Register-Block $reason
    Write-Host ("触发风控 → 降速到 L{0}，暂停 {1} 分钟" -f $st.level, 10 * $st.level)
    exit 3
  }

  if ($LASTEXITCODE -ne 0) {
    Write-Host "[$kw] 采集失败(exit=$LASTEXITCODE)，5 秒后重试一次"
    Start-Sleep -Seconds 5
    $collectOut = node dy_collect.mjs $kw 2>&1

    if ($LASTEXITCODE -ne 0) {
      Write-Host "[$kw] 重试仍失败，跳过该关键词"
      ($collectOut | Out-String) -split "`n" | Select-Object -Last 6 | ForEach-Object { Write-Host "  $_" }
    }
  }

  $file = "E:\ai\data\dy_$kw.json"

  if (-not (Test-Path $file)) {
    Write-Host "[$kw] 采集失败，跳过"
    continue
  }

  $data = Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json

  # 搜索结果为空多半是页面没加载出来，重试一次
  if ([int]$data.count -eq 0) {
    Write-Host "[$kw] 搜索结果为 0，8 秒后重试"
    Start-Sleep -Seconds 8
    node dy_collect.mjs $kw 2>&1 | Out-Null
    if (Test-Path $file) { $data = Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json }
  }

  $ids = @($data.items | Select-Object -First $VideosPerKeyword | ForEach-Object { $_.id })
  Write-Host "[$kw] 共 $($data.count) 个视频，处理前 $($ids.Count) 个"

  $seen = Load-Seen 'douyin'
  Clear-OldSeen 'douyin' $seen | Out-Null

  foreach ($id in $ids) {
    if (Test-SeenRecently 'douyin' $id $seen) {
      Write-Host "  视频 $id 7 天内抓过，跳过"
      $script:dupSkipped++
      continue
    }

    node dy_comments.mjs $id --days $MaxDays --limit $Limit @cityArg @excludeArg 2>&1 | Out-Null

    if ($LASTEXITCODE -eq 3) {
      Write-Host "BLOCKED captcha on video $id"
      $st = Register-Block 'captcha on video'
      Write-Host ("触发风控 → 降速到 L{0}" -f $st.level)
      exit 3
    }

    Set-Seen 'douyin' $id $seen
    Save-Seen 'douyin' $seen

    $cf = "E:\ai\data\dy_comments_$id.json"

    if (Test-Path $cf) {
      try {
        $c = Get-Content $cf -Raw -Encoding UTF8 | ConvertFrom-Json
        $totalComments += [int]$c.stats.raw
        $totalVideos += 1
        Write-Host "  视频 $id 评论 $($c.stats.raw) 条，保留 $($c.stats.kept) 条"
      } catch {}

      $env:DY_KEYWORD = $kw
      node dy_intent.mjs $cf 2>&1 | Out-Null
    }

    Start-Sleep -Seconds (Get-ThrottleDelay 6 12)
  }

  Start-Sleep -Seconds (Get-ThrottleDelay 20 40)
}

$afterLeads = 0
if (Test-Path $leadsFile) {
  try { $afterLeads = (Get-Content $leadsFile -Raw -Encoding UTF8 | ConvertFrom-Json).total } catch { $afterLeads = 0 }
}

Write-Host "本地线索库: $afterLeads 条（新增 $($afterLeads - $beforeLeads)）"

if (-not $SkipPush) {
  Write-Host "推送到后台..."
  node dy_push.mjs 2>&1 | Out-Null
}

Write-Host "检查私信回复..."
node dy_dm_reply.mjs --limit 10 2>&1 | Select-Object -Last 3 | ForEach-Object { Write-Host "  $_" }

Register-Success | Out-Null
Write-Host "采集完成"
Write-Host ("STATS " + (@{ videos = $totalVideos; comments = $totalComments; leads = ($afterLeads - $beforeLeads); skippedDup = [int]$script:dupSkipped } | ConvertTo-Json -Compress))
