param(
    [switch]$TestMode
)

$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$ChecksumsPath = Join-Path $AppDir 'SHA256SUMS.txt'

function Resolve-MessagesPath {
    $fallback = Join-Path $AppDir 'messages.json'
    if (!(Test-Path -LiteralPath $ConfigPath)) { return $fallback }
    try {
        $cfg = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $locale = [string]$cfg.locale
        if ([string]::IsNullOrWhiteSpace($locale)) { $locale = 'ja' }
        if ($locale -notmatch '^[A-Za-z0-9_-]+$') { return $fallback }
        $candidate = Join-Path $AppDir ('locales\' + $locale + '.console.json')
        if (Test-Path -LiteralPath $candidate) { return $candidate }
    } catch {}
    return $fallback
}

$MessagesPath = Resolve-MessagesPath

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
    $limit = 256 - (256 % $chars.Length)
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $sb = New-Object System.Text.StringBuilder
        $buffer = New-Object byte[] 32
        while ($sb.Length -lt $length) {
            $rng.GetBytes($buffer)
            foreach ($b in $buffer) {
                if ($b -lt $limit) {
                    [void]$sb.Append($chars[$b % $chars.Length])
                    if ($sb.Length -ge $length) { break }
                }
            }
        }
        return $sb.ToString()
    } finally { $rng.Dispose() }
}

function Read-ExpectedExecutables {
    if (!(Test-Path -LiteralPath $ChecksumsPath)) {
        throw (Text 'integrityFailed' 'Application integrity check failed.')
    }
    $items = @()
    foreach ($line in (Get-Content -LiteralPath $ChecksumsPath -Encoding UTF8)) {
        if ($line -match '^([0-9a-fA-F]{64})\s\s(.+\.exe)$') {
            $items += [pscustomobject]@{ Hash = $matches[1].ToLowerInvariant(); Name = $matches[2] }
        }
    }
    if ($items.Count -lt 1) { throw (Text 'integrityFailed' 'Application integrity check failed.') }
    return $items
}

function Test-ExecutableHash([string]$path, [string]$expectedHash) {
    if (!(Test-Path -LiteralPath $path)) { return $false }
    try {
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        return ($actual -eq $expectedHash)
    } catch { return $false }
}

function Start-TransferExecutable {
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $ExePath
    $startInfo.WorkingDirectory = $AppDir
    $startInfo.UseShellExecute = $false
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo
    if (!$process.Start()) {
        throw (Text 'startupErrorBody' 'File transfer could not be started.')
    }
    $process.WaitForExit()
    $process.Dispose()
}

if (!(Test-Path -LiteralPath $ConfigPath)) { throw (Text 'configMissing' 'transfer-config.json was not found.') }

$expectedExecutables = @(Read-ExpectedExecutables)
$armItem = $expectedExecutables | Where-Object { $_.Name -like '*_ARM64.exe' } | Select-Object -First 1
$x64Item = $expectedExecutables | Where-Object { $_.Name -notlike '*_ARM64.exe' } | Select-Object -First 1
$arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
$selected = if ($arch -eq 'ARM64' -and $null -ne $armItem) { $armItem } else { $x64Item }
if ($null -eq $selected) { throw (Text 'exeMissing' 'Transfer executable was not found.') }
$ExePath = Join-Path $AppDir $selected.Name
if (!(Test-ExecutableHash $ExePath $selected.Hash)) { throw (Text 'integrityFailed' 'Application integrity check failed.') }

$config = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
$ssid = [string]$config.wifiSsid
$passwordLength = 16
if ($null -ne $config.wifiPasswordLength) { $passwordLength = [int]$config.wifiPasswordLength }
$password = New-SessionPassword $passwordLength
if ([string]::IsNullOrWhiteSpace($ssid) -or $ssid.Length -gt 32) { throw (Text 'invalidSsid' 'The Wi-Fi name is invalid.') }

$env:LOCAL_PHONE_TRANSFER_LAUNCHED = '1'

if ($TestMode) {
    $env:LOCAL_PHONE_TRANSFER_TEST_MODE = '1'
    $env:LOCAL_PHONE_TRANSFER_WIFI_SSID = 'TEST-MODE'
    $env:LOCAL_PHONE_TRANSFER_WIFI_PASSWORD = 'LOCALHOST-ONLY'
    try {
        Write-Host ''
        Write-Host '========================================' -ForegroundColor DarkYellow
        Write-Host (' ' + (Text 'testModeTitle' 'Offline File Transfer - TEST MODE')) -ForegroundColor Yellow
        Write-Host '========================================' -ForegroundColor DarkYellow
        Write-Host ''
        Write-Host (Text 'testModeLocalOnly' 'The test server is available only from this PC (127.0.0.1).') -ForegroundColor Yellow
        Write-Host (Text 'testModeNoWifi' 'Wi-Fi Direct and firewall changes are not used in test mode.') -ForegroundColor DarkGray
        Write-Host ''
        Start-TransferExecutable
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
        Remove-Item Env:LOCAL_PHONE_TRANSFER_TEST_MODE -ErrorAction SilentlyContinue
        Remove-Item Env:LOCAL_PHONE_TRANSFER_WIFI_SSID -ErrorAction SilentlyContinue
        Remove-Item Env:LOCAL_PHONE_TRANSFER_WIFI_PASSWORD -ErrorAction SilentlyContinue
        Remove-Item Env:LOCAL_PHONE_TRANSFER_LAUNCHED -ErrorAction SilentlyContinue
    }
    Write-Host (Text 'finished' 'File transfer has ended.') -ForegroundColor Green
    exit 0
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
    Write-Host (' ' + (Text 'title' 'Offline File Transfer')) -ForegroundColor White
    Write-Host '========================================' -ForegroundColor DarkGray
    Write-Host ''
    Write-Host (Text 'startingWifi' 'Preparing the transfer Wi-Fi...') -ForegroundColor Cyan
    Write-Host ((Text 'wifiNameLabel' 'Wi-Fi name') + ': ' + $ssid)

    $publisher.Start()
    $deadline = (Get-Date).AddSeconds(8)
    while ($publisher.Status.ToString() -eq 'Created' -and (Get-Date) -lt $deadline) { Start-Sleep -Milliseconds 150 }
    if ($publisher.Status.ToString() -ne 'Started') {
        throw ((Text 'wifiStartFailed' 'Could not start the transfer Wi-Fi.') + ' Status=' + $publisher.Status)
    }

    Write-Host (Text 'wifiStarted' 'Transfer Wi-Fi started.') -ForegroundColor Green
    Write-Host (Text 'openingPanel' 'Opening the operation screen...')
    Write-Host (Text 'keepWindowOpen' 'Keep this window open until file transfer is finished.') -ForegroundColor DarkGray
    Write-Host ''
    Start-Sleep -Milliseconds 1000
    Start-TransferExecutable
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
    Remove-Item Env:LOCAL_PHONE_TRANSFER_LAUNCHED -ErrorAction SilentlyContinue
}

Write-Host (Text 'finished' 'File transfer has ended.') -ForegroundColor Green
