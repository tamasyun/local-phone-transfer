$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$MessagesPath = Join-Path $AppDir 'messages.json'

function Load-Messages {
    if (Test-Path -LiteralPath $MessagesPath) {
        try { return Get-Content -LiteralPath $MessagesPath -Raw -Encoding UTF8 | ConvertFrom-Json } catch {}
    }
    return $null
}

$Messages = Load-Messages
function Text([string]$name, [string]$fallback) {
    if ($null -ne $Messages -and $null -ne $Messages.$name) { return [string]$Messages.$name }
    return $fallback
}

function New-SessionPassword([int]$length) {
    if ($length -lt 12) { $length = 12 }
    if ($length -gt 32) { $length = 32 }
    $chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789'
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $bytes = New-Object byte[] $length
        $rng.GetBytes($bytes)
        $sb = New-Object System.Text.StringBuilder
        foreach ($b in $bytes) { [void]$sb.Append($chars[$b % $chars.Length]) }
        return $sb.ToString()
    } finally { $rng.Dispose() }
}

if (!(Test-Path -LiteralPath $ConfigPath)) {
    throw (Text 'configMissing' 'transfer-config.json was not found.')
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
    throw (Text 'exeMissing' 'Transfer executable was not found.')
}

$config = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
$ssid = [string]$config.wifiSsid
$passwordLength = 16
if ($null -ne $config.wifiPasswordLength) { $passwordLength = [int]$config.wifiPasswordLength }
$password = New-SessionPassword $passwordLength

if ([string]::IsNullOrWhiteSpace($ssid) -or $ssid.Length -gt 32) {
    throw (Text 'invalidSsid' 'The Wi-Fi name is invalid.')
}

$env:LOCAL_PHONE_TRANSFER_WIFI_SSID = $ssid
$env:LOCAL_PHONE_TRANSFER_WIFI_PASSWORD = $password

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
    Write-Host '========================================' -ForegroundColor DarkGray
    Write-Host (' ' + (Text 'title' 'Local Phone Transfer')) -ForegroundColor White
    Write-Host '========================================' -ForegroundColor DarkGray
    Write-Host ''
    Write-Host (Text 'startingWifi' 'Preparing the transfer Wi-Fi...') -ForegroundColor Cyan
    Write-Host ((Text 'wifiNameLabel' 'Wi-Fi name') + ': ' + $ssid)

    $publisher.Start()

    $deadline = (Get-Date).AddSeconds(8)
    while ($publisher.Status.ToString() -eq 'Created' -and (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 150
    }

    if ($publisher.Status.ToString() -ne 'Started') {
        throw ((Text 'wifiStartFailed' 'Could not start the transfer Wi-Fi.') + ' Status=' + $publisher.Status)
    }

    Write-Host (Text 'wifiStarted' 'Transfer Wi-Fi started.') -ForegroundColor Green
    Write-Host (Text 'openingPanel' 'Opening the operation screen...')
    Write-Host (Text 'keepWindowOpen' 'Keep this window open until file transfer is finished.') -ForegroundColor DarkGray
    Write-Host ''

    Start-Sleep -Milliseconds 1000
    $process = Start-Process -FilePath $ExePath -WorkingDirectory $AppDir -PassThru
    $process.WaitForExit()
}
catch {
    $detail = $_.Exception.Message
    try {
        Add-Type -AssemblyName PresentationFramework
        $message = (Text 'startupErrorBody' 'File transfer could not be started.') + "`n`n" + $detail
        [System.Windows.MessageBox]::Show($message, (Text 'startupErrorTitle' 'File Transfer - Error'), 'OK', 'Error') | Out-Null
    } catch {}
    Write-Host $detail -ForegroundColor Red
    exit 1
}
finally {
    if ($null -ne $publisher) {
        Write-Host (Text 'stoppingWifi' 'Stopping the transfer Wi-Fi...') -ForegroundColor DarkGray
        try { $publisher.Stop() } catch {}
    }
    Remove-Item Env:LOCAL_PHONE_TRANSFER_WIFI_SSID -ErrorAction SilentlyContinue
    Remove-Item Env:LOCAL_PHONE_TRANSFER_WIFI_PASSWORD -ErrorAction SilentlyContinue
}

Write-Host (Text 'finished' 'File transfer has ended.') -ForegroundColor Green
