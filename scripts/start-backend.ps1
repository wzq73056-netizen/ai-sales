# 启动后端：加载凭据后运行 gva.exe
param([switch]$NoStop)
if (-not $NoStop) { Get-Process gva -ErrorAction SilentlyContinue | Stop-Process -Force; Start-Sleep 2 }
. 'E:\ai\secrets\deepseek.ps1'
Start-Process -FilePath 'E:\ai\gin-vue-admin\server\bin\gva.exe' `
  -RedirectStandardOutput 'E:\ai\gin-vue-admin\server\run.out.log' `
  -RedirectStandardError  'E:\ai\gin-vue-admin\server\run.err.log' `
  -NoNewWindow -WorkingDirectory 'E:\ai\gin-vue-admin\server'
for ($i=0; $i -lt 40; $i++) { Start-Sleep -Milliseconds 500; if ((Test-NetConnection -ComputerName 127.0.0.1 -Port 8888 -WarningAction SilentlyContinue).TcpTestSucceeded) { break } }
"8888: " + (Test-NetConnection -ComputerName 127.0.0.1 -Port 8888 -WarningAction SilentlyContinue).TcpTestSucceeded
