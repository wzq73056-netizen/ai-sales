# 把三处代码汇总到 E:\ai\repo 并推送到 Gitee
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File E:\ai\scripts\git-sync.ps1

$ErrorActionPreference = 'Stop'

$Repo   = 'E:\ai\repo'
$Backend  = 'E:\ai\gin-vue-admin'
$Frontend = 'E:\ai\soybean-admin'
$Scripts  = 'E:\ai\scripts'
$LogDir   = 'E:\ai\data\logs'
$LogFile  = Join-Path $LogDir 'git-sync.log'

if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Force -Path $LogDir | Out-Null }

function Write-Log([string]$Message) {
    $line = "[{0}] {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Message
    Write-Host $line
    Add-Content -Path $LogFile -Value $line -Encoding UTF8
}

function Sync-Dir([string]$Source, [string]$Target, [string[]]$ExcludeDirs) {
    if (-not (Test-Path $Source)) {
        Write-Log "跳过（源不存在）：$Source"
        return
    }

    # 按名字排除：robocopy 会匹配任意层级的同名目录
    $xd = @()
    foreach ($e in $ExcludeDirs) { $xd += @('/XD', $e) }

    $args = @($Source, $Target, '/MIR', '/NFL', '/NDL', '/NJH', '/NJS', '/NP', '/R:1', '/W:1') + $xd
    $args += @('/XF', 'config.yaml', 'config.yaml.bak', '*.log', '*.flag', '*.pid', 'heartbeat*.txt', '*.exe', '*.exe~')

    & robocopy @args | Out-Null

    # robocopy 退出码 0-7 都算成功
    if ($LASTEXITCODE -ge 8) { throw "robocopy 失败，退出码 $LASTEXITCODE（$Source）" }
}

Write-Log '===== 开始同步 ====='

Sync-Dir $Backend  (Join-Path $Repo 'backend')  @('bin', 'node_modules', '.git', ($Backend + '\server\upload'), ($Backend + '\server\log'), ($Backend + '\server\latest_log'), ($Backend + '\web'))
Sync-Dir $Frontend (Join-Path $Repo 'frontend') @('node_modules', 'dist', 'dist-ssr', '.git')
Sync-Dir $Scripts  (Join-Path $Repo 'scripts')  @('node_modules', '__pycache__', '.git')

# 清掉上游项目自带的文档噪声，只留本项目的 README 和 docs
$noise = @(
    'backend\aiDoc',
    'backend\README.md',
    'backend\README-en.md',
    'backend\AGENT.MD',
    'backend\CONTRIBUTING.md',
    'backend\SECURITY.md',
    'backend\CODE_OF_CONDUCT.md',
    'backend\.aone_copilot',
    'backend\.claude',
    'backend\.codex',
    'backend\.cursor',
    'backend\.trae',
    'backend\server\README.md',
    'frontend\README.md',
    'frontend\README.en_US.md',
    'frontend\CHANGELOG.md',
    'frontend\CHANGELOG.zh_CN.md',
    'frontend\AGENTS.md',
    'frontend\docs',
    'frontend\.github'
)

$removed = 0
foreach ($rel in $noise) {
    $full = Join-Path $Repo $rel

    if (Test-Path $full) {
        Remove-Item $full -Recurse -Force -ErrorAction SilentlyContinue
        $removed++
    }
}

if ($removed -gt 0) { Write-Log "已清理上游文档 $removed 处" }

# 配置模板：把真实 config.yaml 脱敏后单独放一份，便于新机器还原
$cfg = Join-Path $Backend 'server\config.yaml'
if (Test-Path $cfg) {
    $safe = Get-Content $cfg -Raw -Encoding UTF8
    $safe = $safe -replace '(password:\s*)\S+', '$1<YOUR_DB_PASSWORD>'
    $safe = $safe -replace '(api-key:\s*)\S+', '$1<YOUR_LLM_KEY>'
    Set-Content -Path (Join-Path $Repo 'backend\server\config.example.yaml') -Value $safe -Encoding UTF8
}

Set-Location $Repo

if (-not (Test-Path (Join-Path $Repo '.git'))) {
    Write-Log '仓库还没初始化，请先执行 docs\git-sync.md 里的首次配置'
    exit 1
}

$status = git status --porcelain
if (-not $status) {
    Write-Log '没有改动，跳过提交'
    exit 0
}

$changed = @($status).Count
git add -A | Out-Null
$msg = 'chore(other): sync ' + $changed + ' files ' + (Get-Date -Format 'yyyy-MM-dd HH:mm')
git commit -m $msg | Out-Null
Write-Log "已提交 $changed 个文件"

$remote = git remote -v
if (-not $remote) {
    Write-Log '没有配置 remote，只完成本地提交'
    exit 0
}

# git 把进度写到 stderr，PowerShell 会当成错误，这里临时放开错误处理
$prev = $ErrorActionPreference
$ErrorActionPreference = 'Continue'

$pushOut = git push origin HEAD 2>&1
$pushCode = $LASTEXITCODE

$ErrorActionPreference = $prev

foreach ($line in $pushOut) { Write-Log ($line -replace ':[0-9a-f]{32}@', ':***@') }

if ($pushCode -ne 0) {
    Write-Log "推送失败，git 退出码 $pushCode"
    exit 1
}

Write-Log '推送成功'