$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$ExePathX64 = Join-Path $AppDir 'スマホファイル転送.exe'
$ExePathArm64 = Join-Path $AppDir 'スマホファイル転送_ARM64.exe'

if (!(Test-Path -LiteralPath $ConfigPath)) {
    throw "transfer-config.json が見つかりません: $ConfigPath"
}

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
$ExePath = if ($arch -eq 'Arm64' -and (Test-Path -LiteralPath $ExePathArm64)) { $ExePathArm64 } else { $ExePathX64 }
if (!(Test-Path -LiteralPath $ExePath)) {
    throw "転送アプリ本体が見つかりません: $ExePath"
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
    Write-Host 'スマホファイル転送を開始しています...' -ForegroundColor Cyan
    Write-Host "  Wi-Fi: $ssid"

    $publisher.Start()

    $deadline = (Get-Date).AddSeconds(8)
    while ($publisher.Status.ToString() -eq 'Created' -and (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 150
    }

    if ($publisher.Status.ToString() -ne 'Started') {
        throw "転送専用Wi-Fiを開始できませんでした。Status=$($publisher.Status)。Windowsのモバイル ホットスポットをOFF、Wi-FiをONにして再試行してください。"
    }

    Write-Host '転送専用Wi-Fiを開始しました。' -ForegroundColor Green
    Write-Host '転送が終わるまで、このウィンドウは閉じないでください。'
    Write-Host ''

    Start-Sleep -Milliseconds 1000
    $process = Start-Process -FilePath $ExePath -WorkingDirectory $AppDir -PassThru
    $process.WaitForExit()
}
catch {
    try {
        Add-Type -AssemblyName PresentationFramework
        [System.Windows.MessageBox]::Show(
            $_.Exception.Message,
            'スマホファイル転送 - 起動エラー',
            'OK',
            'Error'
        ) | Out-Null
    } catch {}
    throw
}
finally {
    if ($null -ne $publisher) {
        try { $publisher.Stop() } catch {}
    }
}
