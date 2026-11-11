# 桌面右下角告警弹窗：必须点「知道了」才关闭
# 由 notify.ps1 调用，不要直接跑

param(
    [Parameter(Mandatory = $true)][string]$Title,
    [string]$Body = ''
)

Add-Type -AssemblyName PresentationFramework, PresentationCore, WindowsBase, System.Windows.Forms

Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32Top {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr hWnd, IntPtr after, int x, int y, int cx, int cy, uint flags);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int cmd);
}
"@

$screen = [System.Windows.Forms.Screen]::PrimaryScreen.WorkingArea

$window = New-Object System.Windows.Window
$window.WindowStyle = 'None'
$window.AllowsTransparency = $true
$window.Background = [System.Windows.Media.Brushes]::Transparent
$window.Topmost = $true
$window.ShowInTaskbar = $true
$window.Title = 'AI 获客告警'
$window.Width = 380
$window.SizeToContent = 'Height'

# 卡片
$border = New-Object System.Windows.Controls.Border
$border.Background = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromRgb(30, 41, 59))
$border.CornerRadius = New-Object System.Windows.CornerRadius 10
$border.BorderBrush = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromRgb(239, 68, 68))
$border.BorderThickness = New-Object System.Windows.Thickness 1
$border.Padding = New-Object System.Windows.Thickness 16, 14, 16, 12

$stack = New-Object System.Windows.Controls.StackPanel

$titleBlock = New-Object System.Windows.Controls.TextBlock
$titleBlock.Text = $Title
$titleBlock.FontSize = 15
$titleBlock.FontWeight = 'SemiBold'
$titleBlock.Foreground = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromRgb(248, 250, 252))
$titleBlock.TextWrapping = 'Wrap'
[void]$stack.Children.Add($titleBlock)

if ($Body) {
    $bodyBlock = New-Object System.Windows.Controls.TextBlock
    $bodyBlock.Text = $Body
    $bodyBlock.FontSize = 12
    $bodyBlock.Margin = New-Object System.Windows.Thickness 0, 8, 0, 0
    $bodyBlock.Foreground = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromRgb(148, 163, 184))
    $bodyBlock.TextWrapping = 'Wrap'
    $bodyBlock.LineHeight = 18
    [void]$stack.Children.Add($bodyBlock)
}

# 按钮：只有点它才关闭
$button = New-Object System.Windows.Controls.Button
$button.Content = '知道了'
$button.Width = 84
$button.Height = 30
$button.Margin = New-Object System.Windows.Thickness 0, 14, 0, 0
$button.HorizontalAlignment = 'Right'
$button.FontSize = 13
$button.Cursor = [System.Windows.Input.Cursors]::Hand
$button.Background = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromRgb(239, 68, 68))
$button.Foreground = [System.Windows.Media.Brushes]::White
$button.BorderThickness = New-Object System.Windows.Thickness 0
$button.Add_Click({ $window.Close() })
[void]$stack.Children.Add($button)

$border.Child = $stack
$window.Content = $border

# 置顶 + 拉前台，避免被全屏窗口盖住
$window.Add_Loaded({
    $handle = (New-Object System.Windows.Interop.WindowInteropHelper $window).Handle

    [void][Win32Top]::ShowWindow($handle, 5)
    [void][Win32Top]::SetWindowPos($handle, [IntPtr](-1), 0, 0, 0, 0, 0x0001 -bor 0x0002)
    [void][Win32Top]::SetForegroundWindow($handle)
    $window.Activate() | Out-Null
    [void]$window.Focus()
})

# 排版完成后再贴到右下角，并记录按钮坐标供验证
$window.Add_ContentRendered({
    $window.Left = $screen.Right - $window.ActualWidth - 20
    $window.Top = $screen.Bottom - $window.ActualHeight - 20

    Start-Sleep -Milliseconds 150

    $btnPoint = $button.PointToScreen((New-Object System.Windows.Point 0, 0))
    $line = 'rendered left={0} top={1} w={2} h={3} btnX={4} btnY={5} btnW={6} btnH={7}' -f `
        [int]$window.Left, [int]$window.Top, [int]$window.ActualWidth, [int]$window.ActualHeight, `
        [int]$btnPoint.X, [int]$btnPoint.Y, [int]$button.ActualWidth, [int]$button.ActualHeight

    Add-Content -Path 'E:\ai\data\logs\toast-diag.log' -Value $line -Encoding UTF8

    # 反复抢一次置顶，避免被全屏窗口压在下面
    $handle2 = (New-Object System.Windows.Interop.WindowInteropHelper $window).Handle
    $script:topTicks = 0
    $topTimer = New-Object System.Windows.Threading.DispatcherTimer
    $topTimer.Interval = [TimeSpan]::FromMilliseconds(400)
    $topTimer.Add_Tick({
        $script:topTicks++

        [void][Win32Top]::SetWindowPos($handle2, [IntPtr](-1), 0, 0, 0, 0, 0x0001 -bor 0x0002 -bor 0x0010)

        if ($script:topTicks -ge 8) { $topTimer.Stop() }
    })
    $topTimer.Start()

    Add-Content -Path 'E:\ai\data\logs\toast-diag.log' -Value ("state visible={0} active={1} topmost={2}" -f $window.IsVisible, $window.IsActive, $window.Topmost) -Encoding UTF8
})

[void]$window.ShowDialog()
