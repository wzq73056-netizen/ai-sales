# Windows 自带 OCR（WinRT）封装：截图找文字位置，用于微信这种没有 UIA 树的窗口
Add-Type -AssemblyName System.Runtime.WindowsRuntime -ErrorAction SilentlyContinue
Add-Type -AssemblyName System.Drawing -ErrorAction SilentlyContinue
Add-Type -AssemblyName System.Windows.Forms -ErrorAction SilentlyContinue

$script:OcrEngine = $null

function Initialize-OcrEngine {
  if ($script:OcrEngine) { return $script:OcrEngine }

  $null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime]
  $null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Foundation, ContentType = WindowsRuntime]
  $null = [Windows.Storage.StorageFile, Windows.Foundation, ContentType = WindowsRuntime]
  $script:OcrEngine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()

  if (-not $script:OcrEngine) {
    throw '系统 OCR 引擎不可用（需要中文语言包）'
  }

  return $script:OcrEngine
}

# WinRT 异步调用需要转成同步
$script:AsTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1'
  })[0]

function Await($operation, $resultType) {
  $task = $script:AsTaskGeneric.MakeGenericMethod($resultType).Invoke($null, @($operation))
  $task.Wait(-1) | Out-Null

  return $task.Result
}

# 从屏幕区域截图（默认整个主屏）
function Save-ScreenShot([string]$path, $bounds) {
  Add-Type -AssemblyName System.Drawing

  if (-not $bounds) {
    Add-Type -AssemblyName System.Windows.Forms
    $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
  }

  $bmp = New-Object System.Drawing.Bitmap $bounds.Width, $bounds.Height
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
  $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
  $g.Dispose()
  $bmp.Dispose()

  return $path
}

# 对图片做 OCR，返回 { Text; Lines = @(@{Text; X; Y; W; H}) }
function Read-ImageText([string]$path) {
  $engine = Initialize-OcrEngine

  $file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) ([Windows.Storage.StorageFile])
  $stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
  $decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
  $bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
  $result = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])

  $lines = @()

  foreach ($line in $result.Lines) {
    $words = @($line.Words)

    if (-not $words.Count) { continue }

    $xs = @($words | ForEach-Object { $_.BoundingRect.X })
    $ys = @($words | ForEach-Object { $_.BoundingRect.Y })
    $rights = @($words | ForEach-Object { $_.BoundingRect.X + $_.BoundingRect.Width })
    $bottoms = @($words | ForEach-Object { $_.BoundingRect.Y + $_.BoundingRect.Height })

    $x = ($xs | Measure-Object -Minimum).Minimum
    $y = ($ys | Measure-Object -Minimum).Minimum
    $right = ($rights | Measure-Object -Maximum).Maximum
    $bottom = ($bottoms | Measure-Object -Maximum).Maximum

    $lines += @{
      Text = $line.Text
      X    = [int]$x
      Y    = [int]$y
      W    = [int]($right - $x)
      H    = [int]($bottom - $y)
      CX   = [int]($x + ($right - $x) / 2)
      CY   = [int]($y + ($bottom - $y) / 2)
    }
  }

  $stream.Dispose()

  return @{ Text = ($lines | ForEach-Object { $_.Text }) -join "`n"; Lines = $lines }
}

# 在 OCR 结果里找包含关键字的行（可多个关键字，按顺序优先）
function Find-TextLine($ocr, [string[]]$keywords) {
  foreach ($kw in $keywords) {
    $hit = $ocr.Lines | Where-Object { $_.Text -like "*$kw*" } | Select-Object -First 1

    if ($hit) { return $hit }
  }

  return $null
}
