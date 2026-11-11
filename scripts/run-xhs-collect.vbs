' 每天静默采集小红书（计划任务调用）
Dim shell
Set shell = CreateObject("WScript.Shell")
shell.Run "powershell -NoProfile -WindowStyle Hidden -NonInteractive -ExecutionPolicy Bypass -File ""E:\ai\scripts\xhs_batch.ps1"" -KeywordsCsv ""鸡蛋批发"" -NotesPerKeyword 3 -Limit 30 -CdpPort 9333", 0, False
