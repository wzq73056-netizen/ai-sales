# 日常维护：日志轮转 + 磁盘监控 + 数据备份
# 建议每天凌晨跑一次（已注册计划任务 DSH-AI-Maintenance）

param(
  [switch]$SkipBackup,
  [switch]$SkipRotate
)

# 从本机 db.json 读取数据库密码（db.json 不进仓库）
function Get-DbPassword {
  $file = 'E:\ai\data\db.json'
  if (-not (Test-Path $file)) { throw "找不到 $file，请参照 docs/db.example.json 创建" }
  return (Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json).password
}
$ErrorActionPreference = 'Continue'

$LOG = 'E:\ai\data\logs\maintenance.log'
$LOG_DIR = 'E:\ai\data\logs'
$ARCHIVE_DIR = 'E:\ai\data\logs\archive'
$BACKUP_DIR = 'E:\ai\backup'
$MYSQL = 'D:\Program Files (x86)\mysql-8.0.26-winx64\bin\mysql.exe'
$MYSQLDUMP = 'D:\Program Files (x86)\mysql-8.0.26-winx64\bin\mysqldump.exe'

if (-not (Test-Path $ARCHIVE_DIR)) { New-Item -ItemType Directory -Path $ARCHIVE_DIR -Force | Out-Null }
if (-not (Test-Path $BACKUP_DIR)) { New-Item -ItemType Directory -Path $BACKUP_DIR -Force | Out-Null }

function Write-Log([string]$msg) {
  $line = "{0}  {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg
  Add-Content -Path $LOG -Value $line -Encoding UTF8
  Write-Host $line
}

# ---- 1. 日志轮转：单文件超过 10MB 归档，保留 7 天 ----
if (-not $SkipRotate) {
  $rotated = 0

  foreach ($file in Get-ChildItem $LOG_DIR -Filter '*.log' -File -ErrorAction SilentlyContinue) {
    if ($file.Length -lt 10MB) { continue }

    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $dest = Join-Path $ARCHIVE_DIR ("{0}-{1}.log" -f $file.BaseName, $stamp)

    try {
      Move-Item $file.FullName $dest -Force
      # 原地留一个新的空文件，避免进程还在往里写
      New-Item -ItemType File -Path $file.FullName -Force | Out-Null
      $rotated++
    } catch {
      Write-Log "日志归档失败 $($file.Name): $($_.Exception.Message)"
    }
  }

  # 清掉 7 天前的归档
  $old = Get-ChildItem $ARCHIVE_DIR -File -ErrorAction SilentlyContinue |
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-7) }
  $oldCount = @($old).Count
  $old | Remove-Item -Force -ErrorAction SilentlyContinue

  Write-Log ("日志轮转完成：归档 {0} 个，清理 {1} 个旧归档" -f $rotated, $oldCount)
}

# ---- 2. 磁盘监控 ----
try {
  $root = [System.IO.Path]::GetPathRoot($LOG).TrimEnd('')
  $driveLetter = $root.Substring(0, 2)   # 例如 E:
  $disk = Get-CimInstance Win32_LogicalDisk -Filter "DeviceID='$driveLetter'"
  $freeGB = [math]::Round($disk.FreeSpace / 1GB, 1)

  Write-Log ("磁盘剩余 {0} GB" -f $freeGB)

  if ($freeGB -lt 15) {
    Write-Log "警告：磁盘不足 15 GB"

    $msg = "磁盘只剩 $freeGB GB，采集和备份可能失败"
    Start-Process -FilePath 'wscript.exe' -ArgumentList @(
      '//B', '//Nologo', 'E:\ai\scripts\run-toast.vbs', $msg
    ) -WindowStyle Hidden -ErrorAction SilentlyContinue
  }
} catch {
  Write-Log "磁盘检查失败: $($_.Exception.Message)"
}

# ---- 3. 数据库 + 状态文件备份 ----
if (-not $SkipBackup) {
  $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
  $tmpDir = Join-Path $BACKUP_DIR "_tmp-$stamp"
  $zip = Join-Path $BACKUP_DIR ("ai-go-$stamp.zip")

  New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

  try {
    $env:MYSQL_PWD = (Get-DbPassword)
    $dumpPath = Join-Path $tmpDir 'ai_go.sql'

    & $MYSQLDUMP -uroot --default-character-set=utf8mb4 --single-transaction --quick --routines=false `
      --databases ai_go --result-file=$dumpPath 2>&1 | Out-Null

    if (-not (Test-Path $dumpPath) -or (Get-Item $dumpPath).Length -lt 1000) {
      throw "mysqldump 输出异常"
    }

    # 关键状态文件（不含明文消息库）
    foreach ($name in @('scrape_throttle.json', 'seen_xhs.json', 'seen_douyin.json', '.content_key')) {
      $src = Join-Path 'E:\ai\data' $name
      if (Test-Path $src) { Copy-Item $src -Destination $tmpDir -Force }
    }

    Compress-Archive -Path (Join-Path $tmpDir '*') -DestinationPath $zip -Force
    Remove-Item $tmpDir -Recurse -Force

    $sizeMB = [math]::Round((Get-Item $zip).Length / 1MB, 2)
    Write-Log ("备份完成：{0}（{1} MB）" -f (Split-Path $zip -Leaf), $sizeMB)
  } catch {
    Write-Log "备份失败: $($_.Exception.Message)"
    if (Test-Path $tmpDir) { Remove-Item $tmpDir -Recurse -Force -ErrorAction SilentlyContinue }
  }

  # 只留最近 7 份
  $keep = Get-ChildItem $BACKUP_DIR -Filter 'ai-go-*.zip' -File |
    Sort-Object LastWriteTime -Descending | Select-Object -Skip 7
  $keepCount = @($keep).Count
  $keep | Remove-Item -Force -ErrorAction SilentlyContinue

  if ($keepCount -gt 0) { Write-Log ("清理旧备份 {0} 份" -f $keepCount) }
}

Write-Log '维护任务结束'
