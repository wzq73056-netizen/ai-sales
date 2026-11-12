# 停止 AI 获客系统的全部进程（后端 / 语音 / 三个监听）
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File E:\ai\scripts\stop-all.ps1

$ErrorActionPreference = 'Continue'

function Stop-ByName([string]$Label, [scriptblock]$Matcher) {
    $procs = Get-CimInstance Win32_Process | Where-Object { & $Matcher $_ }

    if (-not $procs) {
        Write-Host "$Label 未运行"
        return
    }

    foreach ($p in $procs) {
        Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
        Write-Host "已停止 $Label (PID $($p.ProcessId))"
    }
}

Write-Host '=== 停止 AI 获客系统 ==='

# 后端
$gva = Get-Process gva -ErrorAction SilentlyContinue
if ($gva) {
    $gva | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "已停止后端 (PID $($gva.Id -join ','))"
} else {
    Write-Host '后端未运行'
}

# 三个监听
Stop-ByName '微信监听' { $_.Name -eq 'python.exe' -and $_.CommandLine -like '*wx_watch*' }
Stop-ByName '抖音监听' { $_.Name -eq 'node.exe' -and $_.CommandLine -like '*dy_dm_watch*' }
Stop-ByName '小红书监听' { $_.Name -eq 'node.exe' -and $_.CommandLine -like '*xhs_dm_watch*' }

# 语音转写服务（可选，保留的话下次启动不用等模型加载）
$keepAudio = $args -contains '-KeepAudio'

if ($keepAudio) {
    Write-Host '语音服务保留运行'
} else {
    Stop-ByName '语音转写服务' { $_.Name -eq 'python.exe' -and $_.CommandLine -like '*serve_audio*' }
}

Write-Host '=== 完成 ==='
Write-Host '（浏览器里的采集 Edge 窗口没有关闭，需要的话手动关）'
