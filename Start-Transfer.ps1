$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'

if (!(Test-Path -LiteralPath $ConfigPath)) {
    throw "transfer-config.json not found: $ConfigPath"
}

$allExe = Get-ChildItem -LiteralPath $AppDir -Filter '*.exe' -File
$exeArm = $allExe | Where-Object { $_.Name -like '*_ARM64.exe' } | Select-Object -First 1
$exeX64 = $allExe | Where-Object { $_.Name -notlike '*_ARM64.exe' } | Select-Object -First 1

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($arch -eq 'Arm64' -and $null -ne $exeArm) {
    $ExePath = $exeArm.FullName
} elseif ($null -ne $exeX64) {
    $ExePath = $exeX64.FullName
} else {
    throw 'Transfer executable not found.'
}

$config = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
$ssid = [string]$config.wifiSsid
$password = [string]$config.wifiPassword

if ([string]::IsNullOrWhiteSpace($ssid) -or $ssid.Length -gt 32) {
    throw 'wifiSsid must be between 1 and 32 characters.'
}
if ($password.Length -lt 8 -or $password.Length -gt 63) {
    throw 'wifiPassword must be between 8 and 63 characters.'
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
    Write-Host 'Starting local phone transfer...' -ForegroundColor Cyan
    Write-Host "  Wi-Fi: $ssid"

    $publisher.Start()

    $deadline = (Get-Date).AddSeconds(8)
    while ($publisher.Status.ToString() -eq 'Created' -and (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 150
    }

    if ($publisher.Status.ToString() -ne 'Started') {
        throw "Could not start Wi-Fi Direct. Status=$($publisher.Status). Turn off Windows Mobile Hotspot, turn Wi-Fi on, and try again."
    }

    Write-Host 'Local transfer Wi-Fi started.' -ForegroundColor Green
    Write-Host 'Keep this window open until the transfer session ends.'
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
            'Local Phone Transfer - Startup Error',
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
