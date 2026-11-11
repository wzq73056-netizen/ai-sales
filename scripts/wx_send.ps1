# 微信发送（聚焦窗口 → 搜索联系人 → 输入 → 回车 → 还回焦点）
# 用法：
#   .\wx_send.ps1 -Contact "张三" -Message "你好" -DryRun     # 只输入不发送
#   .\wx_send.ps1 -Contact "张三" -Message "你好"             # 真实发送
param(
  [Parameter(Mandatory=$true)][string]$Contact,
  [Parameter(Mandatory=$true)][string]$Message,
  [switch]$DryRun,
  [switch]$Screenshot
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Add-Type -AssemblyName System.Windows.Forms

if (-not ('WxSend' -as [type])) {
  Add-Type -ReferencedAssemblies System.Drawing -Language CSharp -TypeDefinition @'
using System;
using System.Drawing;
using System.Text;
using System.Runtime.InteropServices;

public class WxSend {
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumWindowsProc cb, IntPtr l);
  public delegate bool EnumWindowsProc(IntPtr h, IntPtr l);
  [DllImport("user32.dll")] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr hdc, uint f);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr h);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int c);
  [DllImport("user32.dll")] public static extern void SwitchToThisWindow(IntPtr h, bool alt);
  [DllImport("user32.dll")] public static extern bool AttachThreadInput(uint a, uint b, bool f);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("kernel32.dll")] public static extern uint GetCurrentThreadId();
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr h);
  [DllImport("user32.dll")] public static extern bool GetLastInputInfo(ref LASTINPUTINFO plii);
  [StructLayout(LayoutKind.Sequential)] public struct LASTINPUTINFO { public uint cbSize; public uint dwTime; }

  // 距离上次键鼠操作过了多少毫秒
  public static int IdleMs() {
    var info = new LASTINPUTINFO();
    info.cbSize = (uint)System.Runtime.InteropServices.Marshal.SizeOf(info);
    if (!GetLastInputInfo(ref info)) return 60000;
    return (int)((uint)Environment.TickCount - info.dwTime);
  }
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }

  public static long Find(string title) {
    long found = 0;
    EnumWindows((h, l) => {
      var sb = new StringBuilder(256); GetWindowText(h, sb, 256);
      if (sb.ToString() == title) { found = h.ToInt64(); return false; }
      return true;
    }, IntPtr.Zero);
    return found;
  }

  public static bool Focus(long hwnd) {
    var h = new IntPtr(hwnd);
    uint pid; uint tid = GetWindowThreadProcessId(h, out pid);
    SwitchToThisWindow(h, true);
    System.Threading.Thread.Sleep(120);
    AttachThreadInput(GetCurrentThreadId(), tid, true);
    BringWindowToTop(h);
    SetForegroundWindow(h);
    AttachThreadInput(GetCurrentThreadId(), tid, false);
    System.Threading.Thread.Sleep(220);
    return GetForegroundWindow().ToInt64() == hwnd;
  }

  // 把焦点还给之前的窗口
  public static bool Restore(long hwnd) {
    if (hwnd == 0) return false;
    var h = new IntPtr(hwnd);
    if (IsIconic(h)) return false;
    SetForegroundWindow(h);
    return GetForegroundWindow().ToInt64() == hwnd;
  }

  public static bool Shot(long hwnd, string path) {
    var h = new IntPtr(hwnd); RECT r; GetWindowRect(h, out r);
    using (var bmp = new Bitmap(r.Right - r.Left, r.Bottom - r.Top))
    using (var g = Graphics.FromImage(bmp)) {
      IntPtr hdc = g.GetHdc(); bool ok = PrintWindow(h, hdc, 2); g.ReleaseHdc(hdc);
      bmp.Save(path, System.Drawing.Imaging.ImageFormat.Png); return ok;
    }
  }
}
'@
}

function Send-Text {
  param([string]$Text)
  Set-Clipboard -Value $Text
  Start-Sleep -Milliseconds 180
  [System.Windows.Forms.SendKeys]::SendWait('^v')
  Start-Sleep -Milliseconds 450
}


$hwnd = [WxSend]::Find('微信')
if ($hwnd -eq 0) { Write-Output '{"ok":false,"reason":"找不到微信窗口"}'; exit 2 }

# 记住用户当前在用什么窗口，发完还回去
$previous = ([WxSend]::GetForegroundWindow()).ToInt64()

# 用户正在操作时先等一等，别硬抢（最多等 20 秒）
$waited = 0
while ($waited -lt 20000 -and [WxSend]::IdleMs() -lt 2000) {
  Start-Sleep -Milliseconds 400
  $waited += 400
}

$started = Get-Date

$focused = $false
for ($i = 0; $i -lt 4; $i++) {
  if ([WxSend]::Focus($hwnd)) { $focused = $true; break }
  Start-Sleep -Milliseconds 250
}

if (-not $focused) { Write-Output '{"ok":false,"reason":"无法把微信置前"}'; exit 3 }

try {
  # Ctrl+F 搜索联系人
  [System.Windows.Forms.SendKeys]::SendWait('^f')
  Start-Sleep -Milliseconds 420
  Send-Text -Text $Contact

  Start-Sleep -Milliseconds 460
  [System.Windows.Forms.SendKeys]::SendWait('{ENTER}')
  Start-Sleep -Milliseconds 650

  # 输入消息
  Send-Text -Text $Message

  if ($DryRun) {
    [System.Windows.Forms.SendKeys]::SendWait('^a')
    [System.Windows.Forms.SendKeys]::SendWait('{DEL}')
    if ($Screenshot) { [WxSend]::Shot($hwnd, 'E:\ai\shots\wx-dryrun.png') | Out-Null }
    Write-Output ('{"ok":true,"dryRun":true,"contact":"' + $Contact + '"}')
    exit 0
  }

  [System.Windows.Forms.SendKeys]::SendWait('{ENTER}')
  Start-Sleep -Milliseconds 700
  if ($Screenshot) { [WxSend]::Shot($hwnd, 'E:\ai\shots\wx-last-send.png') | Out-Null }

  # 发送校验：按回车后输入框应该被清空。
  # 之前这里直接报 ok，结果没搜到联系人或没选中会话时，
  # 消息其实还留在输入框里（甚至停留在搜索框），调用方以为发出去了。
  try { Set-Clipboard -Value ' ' -ErrorAction SilentlyContinue } catch { }
  Start-Sleep -Milliseconds 80

  [System.Windows.Forms.SendKeys]::SendWait('^a')
  [System.Windows.Forms.SendKeys]::SendWait('^c')
  Start-Sleep -Milliseconds 220

  $leftover = ''
  try { $leftover = (Get-Clipboard -Raw -ErrorAction SilentlyContinue) } catch { $leftover = '' }

  $normalized = ($Message -replace "`r|`n", ' ').Trim()
  $stillThere = $leftover -and $leftover.Trim().StartsWith($normalized.Substring(0, [Math]::Min(12, $normalized.Length)))

  if ($stillThere) {
    # 清掉残留，避免下次误发
    [System.Windows.Forms.SendKeys]::SendWait('{DEL}')
    Start-Sleep -Milliseconds 120

    $elapsed = [math]::Round(((Get-Date) - $started).TotalSeconds, 1)
    $safe = $Message.Replace('"', '\"')
    Write-Output ('{"ok":false,"reason":"消息还留在输入框，没发出去","contact":"' + $Contact + '","message":"' + $safe + '","seconds":' + $elapsed + '}')
    exit 4
  }

  $elapsed = [math]::Round(((Get-Date) - $started).TotalSeconds, 1)
  Write-Output ('{"ok":true,"dryRun":false,"contact":"' + $Contact + '","message":"' + $Message.Replace('"','\"') + '","seconds":' + $elapsed + '}')
}
finally {
  # 无论成功失败，都尽量把焦点还给用户
  if ($previous -ne 0 -and $previous -ne $hwnd) {
    Start-Sleep -Milliseconds 120
    [WxSend]::Restore($previous) | Out-Null
  }
}
