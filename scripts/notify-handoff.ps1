# 桌面通知（优先用 Windows 原生 Toast，失败回退托盘气泡）
# 用法： .\notify-handoff.ps1 -Title "标题" -Text "内容" [-Seconds 8]
param(
  [Parameter(Mandatory = $true)][string]$Title,
  [Parameter(Mandatory = $true)][string]$Text,
  [int]$Seconds = 8
)

$ErrorActionPreference = 'Stop'

function Send-Toast {
  [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
  [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

  $safeTitle = [System.Security.SecurityElement]::Escape($Title)
  $safeText = [System.Security.SecurityElement]::Escape($Text)

  $xmlString = @"
<toast activationType="foreground" scenario="reminder">
  <visual>
    <binding template="ToastGeneric">
      <text>$safeTitle</text>
      <text>$safeText</text>
    </binding>
  </visual>
</toast>
"@

  $xml = New-Object Windows.Data.Xml.Dom.XmlDocument
  $xml.LoadXml($xmlString)
  $toast = New-Object Windows.UI.Notifications.ToastNotification $xml

  $appId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
  [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)
}

function Send-Balloon {
  Add-Type -AssemblyName System.Windows.Forms
  Add-Type -AssemblyName System.Drawing

  $icon = New-Object System.Windows.Forms.NotifyIcon
  $icon.Icon = [System.Drawing.SystemIcons]::Information
  $icon.BalloonTipTitle = $Title
  $icon.BalloonTipText = $Text
  $icon.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Warning
  $icon.Visible = $true
  $icon.ShowBalloonTip($Seconds * 1000)

  Start-Sleep -Seconds $Seconds
  $icon.Visible = $false
  $icon.Dispose()
}

try {
  Send-Toast
} catch {
  try {
    Send-Balloon
  } catch {
    # 通知失败不影响主流程
  }
}
