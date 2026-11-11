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

# 原生窗口/鼠标操作已预编译成 DLL（后台进程里 Add-Type 会报权限错误）
$dll = 'E:\ai\lib\WxNative.dll'

if (-not ('WxNative' -as [type])) {
  # 用字节加载，避免「Operation is not supported」（文件被标记为外部来源）
  [System.Reflection.Assembly]::Load([System.IO.File]::ReadAllBytes($dll)) | Out-Null
}

# 标题栏/窗口坐标在缩放屏上会错位，先声明 DPI 感知
[WxNative]::MakeDpiAware()

function Normalize([string]$text) { return ($text -replace '\s+', '').ToLower() }

function Get-WechatPid {
  $proc = Get-Process Weixin, WeChat -ErrorAction SilentlyContinue |
    Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1

  if (-not $proc) { throw '微信没在运行' }

  return [uint32]$proc.Id
}

function Find-Window([string]$titleLike) {
  $pidWx = Get-WechatPid

  foreach ($h in [WxNative]::Windows($pidWx)) {
    if ([WxNative]::Title($h) -like $titleLike) { return $h }
  }

  return [IntPtr]::Zero
}

function Focus-Window([IntPtr]$hwnd) {
  [WxNative]::SwitchToThisWindow($hwnd, $true)
  [WxNative]::SetForegroundWindow($hwnd) | Out-Null
  Start-Sleep -Milliseconds 400
}

# 截图 + OCR，坐标换算成屏幕绝对坐标
function Snap-Ocr([IntPtr]$hwnd, [string]$name) {
  $rect = New-Object WxNative+RECT
  [WxNative]::GetWindowRect($hwnd, [ref]$rect) | Out-Null

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

# 在窗口底部找绿色实心按钮（微信的「确定」是白字绿底，OCR 读不到）
function Find-GreenButton([IntPtr]$hwnd, $rect) {
  $w = $rect.Right - $rect.Left
  $h = $rect.Bottom - $rect.Top
  $shot = 'E:\ai\data\tmp\wx_green.png'
  $bounds = New-Object System.Drawing.Rectangle($rect.Left, $rect.Top, $w, $h)
  Save-ScreenShot $shot $bounds | Out-Null

  $bmp = New-Object System.Drawing.Bitmap($shot)
  $rows = @()

  for ($y = [int]($h * 0.75); $y -lt $h; $y++) {
    $cnt = 0; $minX = 99999; $maxX = 0

    for ($x = 0; $x -lt $w; $x++) {
      $c = $bmp.GetPixel($x, $y)
      if ($c.G -gt 140 -and $c.R -lt 120 -and $c.B -lt 160 -and $c.G -gt ($c.R + 40)) {
        $cnt++
        if ($x -lt $minX) { $minX = $x }
        if ($x -gt $maxX) { $maxX = $x }
      }
    }

    if ($cnt -gt 40) { $rows += [pscustomobject]@{ Y = $y; MinX = $minX; MaxX = $maxX } }
  }

  $bmp.Dispose()

  if (-not $rows.Count) { return $null }

  $minY = ($rows | Measure-Object -Property Y -Minimum).Minimum
  $maxY = ($rows | Measure-Object -Property Y -Maximum).Maximum
  $minX = ($rows | Measure-Object -Property MinX -Minimum).Minimum
  $maxX = ($rows | Measure-Object -Property MaxX -Maximum).Maximum

  return [pscustomobject]@{
    Text = '确定(绿底)'
    SX   = [int]($rect.Left + ($minX + $maxX) / 2)
    SY   = [int]($rect.Top + ($minY + $maxY) / 2)
  }
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
$tail = if ($key.Length -ge 4) { $key.Substring($key.Length - 4) } else { $key }

# OCR 对数字常认错（0→O、5→S、6→b），按三级兜底找目标那一行
$target = $ocr.Screen | Where-Object { $_.Norm -match '找手机|查找手机|手机号|QQ号|QQ' } | Select-Object -First 1

if (-not $target) {
  $target = $ocr.Screen | Where-Object { $_.Norm.Contains($tail) } | Select-Object -First 1
}

if (-not $target) {
  $target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) -and $_.Norm -notlike '*搜索结果*' } | Select-Object -First 1
}

if (-not $target) {
  $target = $ocr.Screen | Where-Object { $_.Norm.Contains($key) } | Select-Object -First 1
}

if (-not $target) {
  $result.state = 'not_found'
  $result.detail = '下拉里没有这个号的搜索结果'
  $result | ConvertTo-Json -Compress
  exit 0
}

[WxNative]::Click($target.SX, $target.SY)
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
$notExist = Find-Line $ocrAdd @('不存在', '无法找到', '找不到', '未找到', '账号是否正确', '正确')

if (-not $searchBtn -and $notExist) {
  $result.state = 'not_found'
  $result.detail = '添加朋友窗口提示账号不对/不存在'
  $result | ConvertTo-Json -Compress
  exit 0
}

if ($searchBtn) {
  [WxNative]::Click($searchBtn.SX, $searchBtn.SY)
  Start-Sleep -Seconds 3
  $ocrAdd = Snap-Ocr $addWin 'addwin2'
}

$addBtn = Find-Line $ocrAdd @('添加到通讯录', '添加到通', '加为好友')
$msgBtn = Find-Line $ocrAdd @('发消息')
$notExist2 = Find-Line $ocrAdd @('不存在', '无法找到', '找不到', '未找到', '账号是否正确', '正确')

if ($rateLimited) {
  $result.state = 'rate_limited'
  $result.detail = '微信提示操作过于频繁，需要暂停一段时间'
} elseif ($addBtn) {
  $result.state = 'can_add'

  if (-not $Send) {
    $result.ok = $true
    $result.detail = "可以添加（按钮 $($addBtn.SX),$($addBtn.SY)），未点击"
    $result | ConvertTo-Json -Compress
    exit 0
  }

  [WxNative]::Click($addBtn.SX, $addBtn.SY)
  Start-Sleep -Seconds 2

  # 「申请添加朋友」是独立窗口，必须先点标题栏激活，否则后续点击不生效
  $applyWin = Find-Window '申请添加朋友'

  if ($applyWin -ne [IntPtr]::Zero) {
    $rectApply = New-Object WxNative+RECT
    [WxNative]::GetWindowRect($applyWin, [ref]$rectApply) | Out-Null
    [WxNative]::Click(($rectApply.Left + 180), ($rectApply.Top + 25))
    Start-Sleep -Milliseconds 700
  }

  # 5) 填验证消息
  $ocrDialog = Snap-Ocr $addWin 'dialog'

  if ($Remark) {
    $box = Find-Line $ocrDialog @('我是', '发送添加朋友申请', '验证')

    if ($box) {
      [WxNative]::Click($box.SX, ($box.SY + 26))
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

  # 确定按钮白字绿底，OCR 认不出 → 按绿色像素找按钮中心
  $targetWin = if ($applyWin -ne [IntPtr]::Zero) { $applyWin } else { $addWin }
  $dRect = New-Object WxNative+RECT
  [WxNative]::GetWindowRect($targetWin, [ref]$dRect) | Out-Null

  $sendBtn = Find-GreenButton $targetWin $dRect

  if (-not $sendBtn) {
    # 兜底：左下半区
    $sendBtn = [pscustomobject]@{
      Text = '确定(按位置)'
      SX   = [int]($dRect.Left + 120)
      SY   = [int]($dRect.Bottom - 32)
    }
  }

  [WxNative]::Click($sendBtn.SX, $sendBtn.SY)
  Start-Sleep -Seconds 2

  $after = Snap-Ocr $addWin 'after_send'
  $blocked = Find-Line $after @('操作过于频繁', '过于频繁', '稍后再试')

  if ($blocked) {
    $result.state = 'rate_limited'
    $result.detail = '发送后提示操作过于频繁'
  } else {
    $result.ok = $true
    $result.state = 'request_sent'
    $result.detail = '好友申请已发送'
  }
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
