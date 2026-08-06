$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$ExePath = Join-Path $AppDir 'スマホファイル転送.exe'

if (!(Test-Path -LiteralPath $ConfigPath)) {
    throw "transfer-config.json が見つかりません: $ConfigPath"
}
if (!(Test-Path -LiteralPath $ExePath)) {
    throw "スマホファイル転送.exe が見つかりません: $ExePath"
}

$config = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
$ssid = [string]$config.wifiSsid
$password = [string]$config.wifiPassword

if ([string]::IsNullOrWhiteSpace($ssid) -or $ssid.Length -gt 32) {
    throw 'wifiSsid は1〜32文字で指定してください。'
}
if ($password.Length -lt 8 -or $password.Length -gt 63) {
    throw 'wifiPassword は8〜63文字で指定してください。'
}

# Windows PowerShell 5.1 は Windows Runtime (WinRT) を直接利用できます。
# Wi-Fi Direct Legacy mode を有効にすると、iPhone/Android からは通常の
# WPA2 Wi-Fiアクセスポイントとして見えます。インターネット接続は不要です。
Add-Type -AssemblyName System.Runtime.WindowsRuntime

$PublisherType = [Windows.Devices.WiFiDirect.WiFiDirectAdvertisementPublisher, Windows.Devices.WiFiDirect, ContentType=WindowsRuntime]
$CredentialType = [Windows.Security.Credentials.PasswordCredential, Windows.Security.Credentials, ContentType=WindowsRuntime]

$publisher = $null
try {
    $publisher = [Activator]::CreateInstance($PublisherType)
    $advertisement = $publisher.Advertisement
    $advertisement.IsAutonomousGroupOwnerEnabled = $true

    $legacy = $advertisement.LegacySettings
    $legacy.IsEnabled = $true
    $legacy.Ssid = $ssid

    $credential = [Activator]::CreateInstance($CredentialType)
    $credential.Password = $password
    $legacy.Passphrase = $credential

    Write-Host ''
    Write-Host '転送専用Wi-Fiを開始しています...' -ForegroundColor Cyan
    Write-Host "  SSID: $ssid"

    $publisher.Start()

    $deadline = (Get-Date).AddSeconds(8)
    while ($publisher.Status.ToString() -eq 'Created' -and (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 150
    }

    if ($publisher.Status.ToString() -ne 'Started') {
        throw "Wi-Fi Directを開始できませんでした。Status=$($publisher.Status)。Windowsのモバイル ホットスポットがONならOFFにし、Wi-FiをONにして再試行してください。"
    }

    Write-Host '転送専用Wi-Fiを開始しました。' -ForegroundColor Green
    Write-Host 'このPowerShellウィンドウは転送終了まで閉じないでください。'
    Write-Host ''

    # Wi-Fi Direct仮想アダプターにIPv4が付くまで少し待ってから本体を起動します。
    Start-Sleep -Milliseconds 800
    $process = Start-Process -FilePath $ExePath -WorkingDirectory $AppDir -PassThru
    $process.WaitForExit()
}
catch {
    Add-Type -AssemblyName PresentationFramework
    [System.Windows.MessageBox]::Show(
        $_.Exception.Message,
        'ルーターなしファイル転送 - 起動エラー',
        'OK',
        'Error'
    ) | Out-Null
    throw
}
finally {
    if ($null -ne $publisher) {
        try { $publisher.Stop() } catch {}
    }
}
