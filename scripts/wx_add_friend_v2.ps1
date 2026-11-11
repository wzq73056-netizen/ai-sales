# 微信加好友（桌面端 4.x，无 UIA 树 → 截图 + 系统 OCR 找按钮）
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File wx_add_friend.ps1 -Contact "13800001111" -Remark "你好" -Send
#   不带 -Send 只探测，不点击
#
# 返回 JSON： state = can_add / already_friend / not_found / unknown
param(
  [Parameter(Mandatory = $true)][string]$Contact,
  [string]$Remark = '',
  [switch]$Send
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

. 'E:\ai\scripts\ocr.ps1'

Add-Type @'
using System;
using System.Runtime.InteropServices;

public class WxFriend {
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint f, uint dx, uint dy, uint d, UIntPtr e);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern void SwitchToThisWindow(IntPtr h, bool fAltTab);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();

  [StructLayout(LayoutKind.Sequential)]
  public struct RECT { public int Left, Top, Right, Bottom; }

  public static void Click(int x, int y) {
    SetCursorPos(x, y);
    System.Threading.Thread.Sleep(120);
    mouse_event(0x0002, 0, 0, 0, UIntPtr.Zero);
    System.Threading.Thread.Sleep(60);
    mouse_event(0x0004, 0, 0, 0, UIntPtr.Zero);
  }
}
'@

function Normalize([string]$text) { return ($text -replace '\s+', '').ToLower() }

function Get-WechatWindow {
  $proc = Get-Process Weixin, WeChat -ErrorAction SilentlyContinue |
    Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1

  if (-not $proc) { throw '微信没在运行' }

  return $proc.MainWindowHandle
}

function Focus-Wechat([IntPtr]$hwnd) {
  [WxFriend]::SwitchToThisWindow($hwnd, $true)
  [WxFriend]::SetForegroundWindow($hwnd) | Out-Null
  Start-Sleep -Milliseconds 500

  return ([WxFriend]::GetForegroundWindow() -eq $hwnd)
}

function Get-Bounds([IntPtr]$hwnd) {
  $rect = New-Object WxFriend+RECT
  [WxFriend]::GetWindowRect($hwnd, [ref]$rect) | Out-Null

  return @{
    Left   = $rect.Left
    Top    = $rect.Top
    Rect   = New-Object System.Drawing.Rectangle($rect.Left, $rect.Top, ($rect.Right - $rect.Left), ($rect.Bottom - $rect.Top))
    Width  = $rect.Right - $rect.Left
    Height = $rect.Bottom - $rect.Top
  }
}

$global:WxBounds = $null

function Snap-Ocr([string]$name) {
  $shot = "E:\ai\data\tmp\wx_$name.png"
  Save-ScreenShot $shot $global:WxBounds.Rect | Out-Null

  $ocr = Read-ImageText $shot
  $screen = @()

  foreach ($line in $ocr.Lines) {
    $screen += [pscustomobject]@{
      Text = $line.Text
      Norm = Normalize $line.Text
      SX   = $global:WxBounds.Left + $line.CX
      SY   = $global:WxBounds.Top + $line.CY
    }
  }

  $ocr | Add-Member -NotePropertyName Screen -NotePropertyValue $screen -Force

  return $ocr
}

function Find-Line($ocr, [string[]]$keywords) {
  foreach ($kw in $keywords) {
    $k = Normalize $kw
    $hit = $ocr.Screen | Where-Object { $_.Norm.Contains($k) } | Select-Object -First 1

    if ($hit) { return $hit }
  }

  return $null
}

$result = [ordered]@{ ok = $false; contact = $Contact; state = 'unknown'; detail = '' }

$hwnd = Get-WechatWindow

if (-not (Focus-Wechat $hwnd)) {
  $result.detail = '微信窗口没切到前台'
  $result | ConvertTo-Json -Compress
  exit 1
}

$global:WxBounds = Get-Bounds $hwnd

# 1) 搜索框输入号码
$wsh = New-Object -ComObject WScript.Shell
$wsh.SendKeys('^f') | Out-Null
Start-Sleep -Milliseconds 400
$wsh.SendKeys('^a') | Out-Null
$wsh.SendKeys('{DEL}') | Out-Null
Start-Sleep -Milliseconds 200

Set-Clipboard -Value $Contact
Start-Sleep -Milliseconds 150
$wsh.SendKeys('^v') | Out-Null
Start-Sleep -Milliseconds 1200

# 2) 点下拉里的搜索结果
$ocr = Snap-Ocr 'search'
$key = Normalize $Contact

$target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) -and -not $_.Norm.Contains('搜索') } | Select-Object -First 1

if (-not $target) {
  $target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) } | Select-Object -First 1
}

if (-not $target) {
  $result.state = 'not_found'
  $result.detail = '搜索结果里没有这个号'
  $result | ConvertTo-Json -Compress
  exit 0
}

[WxFriend]::Click($target.SX, $target.SY)
Start-Sleep -Seconds 4

# 3) 判定状态
$ocr2 = Snap-Ocr 'profile'

$addBtn = Find-Line $ocr2 @('添加到通讯录', '添加到通', '加为好友')
$msgBtn = Find-Line $ocr2 @('发消息')
$notFound = Find-Line $ocr2 @('不存在', '未找到', '无法找到', '没有找到')

if ($addBtn) {
  $result.state = 'can_add'

  if (-not $Send) {
    $result.ok = $true
    $result.detail = "可添加（按钮 $($addBtn.SX),$($addBtn.SY)），未点击"
    $result | ConvertTo-Json -Compress
    exit 0
  }

  [WxFriend]::Click($addBtn.SX, $addBtn.SY)
  Start-Sleep -Seconds 2

  $ocr3 = Snap-Ocr 'dialog'
  $sendBtn = Find-Line $ocr3 @('发送')

  if (-not $sendBtn) {
    $result.detail = '点了添加到通讯录，但没出现发送按钮'
    $result | ConvertTo-Json -Compress
    exit 2
  }

  if ($Remark) {
    $box = Find-Line $ocr3 @('我是', '发送添加朋友申请', '你好')

    if ($box) {
      [WxFriend]::Click($box.SX, ($box.SY + 26))
      Start-Sleep -Milliseconds 300
      $wsh.SendKeys('^a') | Out-Null
      $wsh.SendKeys('{DEL}') | Out-Null
      Set-Clipboard -Value $Remark
      Start-Sleep -Milliseconds 150
      $wsh.SendKeys('^v') | Out-Null
      Start-Sleep -Milliseconds 500
    }

    $ocr4 = Snap-Ocr 'dialog2'
    $sendBtn2 = Find-Line $ocr4 @('发送')

    if ($sendBtn2) { $sendBtn = $sendBtn2 }
  }

  [WxFriend]::Click($sendBtn.SX, $sendBtn.SY)
  Start-Sleep -Seconds 2

  $result.ok = $true
  $result.state = 'request_sent'
  $result.detail = '好友申请已发送'
} elseif ($msgBtn) {
  $result.ok = $true
  $result.state = 'already_friend'
  $result.detail = '已经是好友'
} elseif ($notFound) {
  $result.state = 'not_found'
  $result.detail = '没有这个用户'
} else {
  $result.detail = '页面文字没认出来：' + (($ocr2.Screen | Select-Object -First 6 | ForEach-Object { $_.Text }) -join ' / ')
}

$result | ConvertTo-Json -Compress
