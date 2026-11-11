# 主动通知：微信掉线、服务异常时推到手机
# 配置：E:\ai\data\notify.json
#   { "serverchan": "SCT开头的SendKey", "webhook": "企业微信/钉钉机器人地址", "cooldown_min": 30 }
#
# 用法：powershell -NoProfile -File E:\ai\scripts\notify.ps1 -Title "微信掉线" -Body "需要重新扫码登录"

param(
    [Parameter(Mandatory = $true)][string]$Title,
    [string]$Body = '',
    [switch]$Force
)

$ErrorActionPreference = 'Continue'

$ConfigPath = 'E:\ai\data\notify.json'
$StatePath  = 'E:\ai\data\notify_state.json'
$LogFile    = 'E:\ai\data\logs\notify.log'

$logDir = Split-Path $LogFile -Parent
if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Force -Path $logDir | Out-Null }

function Write-Log([string]$Message) {
    $line = "[{0}] {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Message
    Add-Content -Path $LogFile -Value $line -Encoding UTF8
}

$config = @{ serverchan = ''; webhook = ''; cooldown_min = 30 }
if (Test-Path $ConfigPath) {
    try {
        $loaded = Get-Content $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($loaded.serverchan) { $config.serverchan = [string]$loaded.serverchan }
        if ($loaded.webhook)     { $config.webhook = [string]$loaded.webhook }
        if ($loaded.cooldown_min) { $config.cooldown_min = [int]$loaded.cooldown_min }
    } catch {
        Write-Log "配置文件解析失败：$($_.Exception.Message)"
    }
}

# 冷却：同一条消息 30 分钟内只推一次，避免每 2 分钟的健康检查刷屏
$now = Get-Date
# 冷却记录：{ "标题": "时间戳" }，5.1 不支持 -AsHashtable，用 PSCustomObject 处理
$state = [ordered]@{}
if (Test-Path $StatePath) {
    try {
        $loadedState = Get-Content $StatePath -Raw -Encoding UTF8 | ConvertFrom-Json
        foreach ($prop in $loadedState.PSObject.Properties) { $state[$prop.Name] = [string]$prop.Value }
    } catch {
        $state = [ordered]@{}
    }
}

$key = $Title
if (-not $Force -and $state.Contains($key)) {
    $last = [datetime]::Parse($state[$key])
    $elapsed = ($now - $last).TotalMinutes

    if ($elapsed -lt $config.cooldown_min) {
        Write-Log "冷却中（$([math]::Round($elapsed,0)) 分钟前已推送过「$Title」），跳过"
        exit 0
    }
}

$sent = $false

# ① Server 酱：微信服务号推送，手机上直接收到
if ($config.serverchan) {
    try {
        $url = "https://sctapi.ftqq.com/$($config.serverchan).send"
        $resp = Invoke-RestMethod -Uri $url -Method Post -Body @{ title = $Title; desp = $Body } -TimeoutSec 15
        if ($resp.code -eq 0) {
            Write-Log "Server酱推送成功：$Title"
            $sent = $true
        } else {
            Write-Log "Server酱返回异常：$($resp.message)"
        }
    } catch {
        Write-Log "Server酱推送失败：$($_.Exception.Message)"
    }
}

# ② 群机器人（企业微信/钉钉）：填 webhook 地址即可
if ($config.webhook) {
    try {
        $payload = @{ msgtype = 'text'; text = @{ content = "$Title`n$Body" } } | ConvertTo-Json -Depth 4
        Invoke-RestMethod -Uri $config.webhook -Method Post -ContentType 'application/json' -Body $payload -TimeoutSec 15 | Out-Null
        Write-Log "Webhook 推送成功：$Title"
        $sent = $true
    } catch {
        Write-Log "Webhook 推送失败：$($_.Exception.Message)"
    }
}

# ③ 桌面弹窗：不依赖任何账号，默认就显示
$toastScript = 'E:\ai\scripts\desktop-toast.ps1'
if (Test-Path $toastScript) {
    try {
        $args = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $toastScript, '-Title', $Title)
        if ($Body) { $args += @('-Body', $Body) }
        Start-Process -FilePath 'powershell.exe' -ArgumentList $args -WindowStyle Hidden | Out-Null
        Write-Log "桌面弹窗已显示：$Title"
        $sent = $true
    } catch {
        Write-Log "桌面弹窗失败：$($_.Exception.Message)"
    }
}

if (-not $sent) {
    Write-Log "没有可用的通知方式，消息只写进日志：$Title | $Body"
}

$state[$key] = $now.ToString('o')
try { $state | ConvertTo-Json -Depth 4 | Set-Content -Path $StatePath -Encoding UTF8 } catch { }

exit 0
