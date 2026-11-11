' 静默运行 health-check.ps1（计划任务用，避免每次弹黑框）
Dim shell, cmd
Set shell = CreateObject("WScript.Shell")
cmd = "powershell -NoProfile -WindowStyle Hidden -NonInteractive -ExecutionPolicy Bypass -File ""E:\ai\scripts\health-check.ps1"""
shell.Run cmd, 0, False
