# 微信加好友（桌面端 4.x）
#
# 微信没有可用 UIA 树 → 用「窗口截图 + Windows 自带 OCR」定位按钮坐标。
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File wx_add_friend.ps1 -Contact "13800001111" -Remark "你好，谈鸡蛋批发" -Send
#   不加 -Send：只走到「找到可添加」为止，不点任何按钮
#
# 输出 JSON：
#   state = can_add / request_sent / already_friend / not_found / unknown

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
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public class WxWin {
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumWindowsProc f, IntPtr l);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowTextW(IntPtr h, StringBuilder s, int c);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint f, uint dx, uint dy, uint d, UIntPtr e);
  [DllImport("user32.dll")] public static extern void SwitchToThisWindow(IntPtr h, bool alt);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();

  public delegate bool EnumWindowsProc(IntPtr h, IntPtr l);

  [StructLayout(LayoutKind.Sequential)]
  public struct RECT { public int Left, Top, Right, Bottom; }

  public static List<IntPtr> Windows(uint pid) {
    var list = new List<IntPtr>();
    EnumWindows((h, l) => {
      uint p; GetWindowThreadProcessId(h, out p);
      if (p == pid && IsWindowVisible(h)) list.Add(h);
      return true;
    }, IntPtr.Zero);
    return list;
  }

  public static string Title(IntPtr h) {
    var sb = new StringBuilder(256);
    GetWindowTextW(h, sb, 256);
    return sb.ToString();
  }

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

function Get-WechatPid {
  $proc = Get-Process Weixin, WeChat -ErrorAction SilentlyContinue |
    Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1

  if (-not $proc) { throw '微信没在运行' }

  return [uint32]$proc.Id
}

function Find-Window([string]$titleLike) {
  $pidWx = Get-WechatPid

  foreach ($h in [WxWin]::Windows($pidWx)) {
    if ([WxWin]::Title($h) -like $titleLike) { return $h }
  }

  return [IntPtr]::Zero
}

function Focus-Window([IntPtr]$hwnd) {
  [WxWin]::SwitchToThisWindow($hwnd, $true)
  [WxWin]::SetForegroundWindow($hwnd) | Out-Null
  Start-Sleep -Milliseconds 400
}

# 截图 + OCR，坐标换算成屏幕绝对坐标
function Snap-Ocr([IntPtr]$hwnd, [string]$name) {
  $rect = New-Object WxWin+RECT
  [WxWin]::GetWindowRect($hwnd, [ref]$rect) | Out-Null

  $bounds = New-Object System.Drawing.Rectangle($rect.Left, $rect.Top, ($rect.Right - $rect.Left), ($rect.Bottom - $rect.Top))
  $shot = "E:\ai\data\tmp\wx_$name.png"
  Save-ScreenShot $shot $bounds | Out-Null

  $ocr = Read-ImageText $shot
  $screen = @()

  foreach ($line in $ocr.Lines) {
    $screen += [pscustomobject]@{
      Text = $line.Text
      Norm = Normalize $line.Text
      SX   = $rect.Left + $line.CX
      SY   = $rect.Top + $line.CY
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

$main = Find-Window '微信'

if ($main -eq [IntPtr]::Zero) {
  $result.detail = '找不到微信主窗口'
  $result | ConvertTo-Json -Compress
  exit 1
}

Focus-Window $main
$wsh = New-Object -ComObject WScript.Shell

# 1) 搜索框里输入号码
$wsh.SendKeys('^f') | Out-Null
Start-Sleep -Milliseconds 400
$wsh.SendKeys('^a') | Out-Null
$wsh.SendKeys('{DEL}') | Out-Null
Start-Sleep -Milliseconds 200
Set-Clipboard -Value $Contact
Start-Sleep -Milliseconds 150
$wsh.SendKeys('^v') | Out-Null
Start-Sleep -Milliseconds 1300

# 2) 点下拉里的「网络查找…」那一条
$ocr = Snap-Ocr $main 'search'
$key = Normalize $Contact

$target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) -and $_.Norm -notlike '*搜索*' } | Select-Object -First 1

if (-not $target) {
  $target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) } | Select-Object -First 1
}

if (-not $target) {
  $result.state = 'not_found'
  $result.detail = '下拉里没有这个号的搜索结果'
  $result | ConvertTo-Json -Compress
  exit 0
}

[WxWin]::Click($target.SX, $target.SY)
Start-Sleep -Seconds 3

# 3) 找到「添加朋友」窗口
$addWin = Find-Window '添加朋友'

if ($addWin -eq [IntPtr]::Zero) {
  # 没有弹窗 → 搜到的是已有好友，直接开了聊天
  $result.ok = $true
  $result.state = 'already_friend'
  $result.detail = '没有弹添加朋友窗口，说明已经是好友'
  $result | ConvertTo-Json -Compress
  exit 0
}

Focus-Window $addWin
$ocrAdd = Snap-Ocr $addWin 'addwin'

# 4) 点窗口里的「搜索」按钮
$searchBtn = Find-Line $ocrAdd @('搜索')
$alreadyFriend = Find-Line $ocrAdd @('发消息')
$notExist = Find-Line $ocrAdd @('不存在', '账号是否正确', '未找到')

if (-not $searchBtn -and $notExist) {
  $result.state = 'not_found'
  $result.detail = '添加朋友窗口提示账号不对/不存在'
  $result | ConvertTo-Json -Compress
  exit 0
}

if ($searchBtn) {
  [WxWin]::Click($searchBtn.SX, $searchBtn.SY)
  Start-Sleep -Seconds 3
  $ocrAdd = Snap-Ocr $addWin 'addwin2'
}

$addBtn = Find-Line $ocrAdd @('添加到通讯录', '添加到通', '加为好友')
$msgBtn = Find-Line $ocrAdd @('发消息')
$notExist2 = Find-Line $ocrAdd @('不存在', '账号是否正确', '未找到')

if ($addBtn) {
  $result.state = 'can_add'

  if (-not $Send) {
    $result.ok = $true
    $result.detail = "可以添加（按钮 $($addBtn.SX),$($addBtn.SY)），未点击"
    $result | ConvertTo-Json -Compress
    exit 0
  }

  [WxWin]::Click($addBtn.SX, $addBtn.SY)
  Start-Sleep -Seconds 2

  # 5) 填验证消息
  $ocrDialog = Snap-Ocr $addWin 'dialog'

  if ($Remark) {
    $box = Find-Line $ocrDialog @('我是', '发送添加朋友申请', '验证')

    if ($box) {
      [WxWin]::Click($box.SX, ($box.SY + 26))
      Start-Sleep -Milliseconds 300
      $wsh.SendKeys('^a') | Out-Null
      $wsh.SendKeys('{DEL}') | Out-Null
      Set-Clipboard -Value $Remark
      Start-Sleep -Milliseconds 150
      $wsh.SendKeys('^v') | Out-Null
      Start-Sleep -Milliseconds 500
      $ocrDialog = Snap-Ocr $addWin 'dialog2'
    }
  }

  $sendBtn = Find-Line $ocrDialog @('发送', '确定')

  if (-not $sendBtn) {
    $result.detail = '没找到发送按钮'
    $result | ConvertTo-Json -Compress
    exit 2
  }

  [WxWin]::Click($sendBtn.SX, $sendBtn.SY)
  Start-Sleep -Seconds 2

  $result.ok = $true
  $result.state = 'request_sent'
  $result.detail = '好友申请已发送'
} elseif ($msgBtn) {
  $result.ok = $true
  $result.state = 'already_friend'
  $result.detail = '添加朋友窗口显示可发消息，已是好友'
} elseif ($notExist2) {
  $result.state = 'not_found'
  $result.detail = '该账号不存在或不可添加'
} else {
  $result.detail = '认不出界面：' + (($ocrAdd.Screen | Select-Object -First 8 | ForEach-Object { $_.Text }) -join ' / ')
}

$result | ConvertTo-Json -Compress
