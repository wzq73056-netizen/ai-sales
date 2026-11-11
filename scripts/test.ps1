# 一键自测：Go 单测 + 脚本语法 + 页面编译 + 接口冒烟
# 用法： powershell -ExecutionPolicy Bypass -File E:\ai\scripts\test.ps1

$ErrorActionPreference = 'Continue'

$results = @()

function Add-Result([string]$name, [bool]$ok, [string]$detail = '') {
  $script:results += [pscustomobject]@{ 项目 = $name; 结果 = if ($ok) { 'PASS' } else { 'FAIL' }; 说明 = $detail }
}

Write-Host '=== 1. Go 单元测试 ===' -ForegroundColor Cyan
$env:Path += ';D:\Program Files\go\bin'
$env:GOPROXY = 'https://goproxy.cn,direct'

Push-Location 'E:\ai\gin-vue-admin\server'
$goOut = & go test ./service/lead/... 2>&1
$goOk = $LASTEXITCODE -eq 0
Pop-Location

Add-Result 'Go 单测 (service/lead)' $goOk ($goOut | Select-Object -Last 1)

Write-Host '=== 2. Node 脚本语法 ===' -ForegroundColor Cyan
$node = 'C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\node\bin\node.exe'
if (-not (Test-Path $node)) { $node = 'node' }

$badScripts = @()

foreach ($file in Get-ChildItem 'E:\ai\scraper\*.mjs' -File) {
  & $node --check $file.FullName 2>$null | Out-Null

  if ($LASTEXITCODE -ne 0) { $badScripts += $file.Name }
}

Add-Result ("Node 脚本 ({0} 个)" -f (Get-ChildItem 'E:\ai\scraper\*.mjs' -File).Count) ($badScripts.Count -eq 0) ($badScripts -join ', ')

Write-Host '=== 3. Python 脚本语法 ===' -ForegroundColor Cyan
$py = 'C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\python\python.exe'
$pyOk = $true
$pyDetail = ''

foreach ($file in @('E:\ai\scripts\wx_watch.py')) {
  if (-not (Test-Path $file)) { continue }

  $out = & $py -m py_compile $file 2>&1

  if ($LASTEXITCODE -ne 0) {
    $pyOk = $false
    $pyDetail = $out | Select-Object -Last 1
  }
}

Add-Result 'Python 脚本 (wx_watch.py)' $pyOk $pyDetail

Write-Host '=== 4. Vue 页面编译 ===' -ForegroundColor Cyan
$checkScript = @'
import { parse } from '@vue/compiler-sfc';
import fs from 'node:fs';
import path from 'node:path';

const root = 'E:/ai/soybean-admin/src/views';
const dirs = ['workbench', 'inbox', 'lead', 'wechat', 'douyin', 'xhs', 'analytics'];
let bad = 0;

for (const dir of dirs) {
  const file = path.join(root, dir, 'index.vue');
  if (!fs.existsSync(file)) { console.log(`MISSING ${dir}`); bad++; continue; }
  const { errors } = parse(fs.readFileSync(file, 'utf-8'), { filename: file });
  if (errors.length) { console.log(`FAIL ${dir}: ${errors[0].message}`); bad++; }
}

console.log(bad === 0 ? 'ALL_OK' : `BAD_${bad}`);
process.exit(bad === 0 ? 0 : 1);
'@

$checkScript | Out-File 'E:\ai\soybean-admin\_sfc_check.mjs' -Encoding utf8
Push-Location 'E:\ai\soybean-admin'
$sfcOut = & $node '_sfc_check.mjs' 2>&1
$sfcOk = $LASTEXITCODE -eq 0
Pop-Location
Remove-Item 'E:\ai\soybean-admin\_sfc_check.mjs' -Force -ErrorAction SilentlyContinue

Add-Result 'Vue 页面编译 (7 个)' $sfcOk ($sfcOut | Select-Object -Last 1)

Write-Host '=== 4b. 前端单元测试 (vitest) ===' -ForegroundColor Cyan
Push-Location 'E:\ai\soybean-admin'
$vitestOut = & 'E:\ai\soybean-admin\node_modules\.bin\vitest.CMD' run -c vitest.config.ts 2>&1
$vitestOk = $LASTEXITCODE -eq 0
Pop-Location

Add-Result '前端单测 (vitest)' $vitestOk (($vitestOut | Select-String -Pattern 'Tests\s+\d+ passed' | Select-Object -First 1).Line)

Write-Host '=== 5. 接口冒烟 ===' -ForegroundColor Cyan
$smokeOk = $true
$smokeDetail = ''

try {
  $login = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/base/login' -Method Post `
    -Body (@{ username = 'admin'; password = '123456'; captcha = ''; captchaId = '' } | ConvertTo-Json) `
    -ContentType 'application/json' -TimeoutSec 15

  if ($login.code -ne 0) { throw '登录失败' }

  $headers = @{ 'x-token' = $login.data.token }

  $endpoints = @(
    'lead/getWorkbench',
    'lead/getInbox?limit=5',
    'lead/getLeadList?page=1&pageSize=5',
    'lead/getScriptStats',
    'lead/getXhsRuns',
    'lead/getScrapeKeywords',
    'lead/getScrapeBlacklist',
    'lead/getXhsSchedule',
    'lead/getWxAddConfig',
    'lead/getWxAddQueue',
    'lead/getFollowupReminders',
    'lead/getWxCustomerList?page=1&pageSize=5&minScore=0',
    'lead/getLeadAnalytics'
  )

  $failed = @()

  foreach ($path in $endpoints) {
    try {
      $res = Invoke-RestMethod -Uri "http://127.0.0.1:8888/$path" -Headers $headers -TimeoutSec 20
      if ($res.code -ne 0) { $failed += "$path($($res.msg))" }
    } catch {
      $failed += $path
    }
  }

  if ($failed.Count) {
    $smokeOk = $false
    $smokeDetail = $failed -join ', '
  } else {
    $smokeDetail = "$($endpoints.Count) 个接口正常"
  }
} catch {
  $smokeOk = $false
  $smokeDetail = $_.Exception.Message
}

Add-Result '接口冒烟' $smokeOk $smokeDetail

Write-Host ''
Write-Host '=== 结果 ===' -ForegroundColor Cyan
$results | Format-Table -AutoSize

$fail = @($results | Where-Object { $_.结果 -eq 'FAIL' }).Count

if ($fail -gt 0) {
  Write-Host "有 $fail 项失败" -ForegroundColor Red
  exit 1
}

Write-Host '全部通过' -ForegroundColor Green
exit 0
