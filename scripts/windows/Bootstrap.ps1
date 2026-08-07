param(
    [switch]$TestMode,
    [switch]$SkipUpdate
)

$ErrorActionPreference = 'SilentlyContinue'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$ChecksumsPath = Join-Path $AppDir 'SHA256SUMS.txt'
$VersionPath = Join-Path $AppDir 'VERSION.txt'
$UpdaterPath = Join-Path $AppDir 'Update.ps1'
$FirewallHelperPath = Join-Path $AppDir 'Configure-Firewall.ps1'
$FirewallMarkerPath = Join-Path $AppDir '.firewall-configured'
$LatestReleaseApi = 'https://api.github.com/repos/tamasyun/local-phone-transfer/releases/latest'

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

function Text([string]$name, [string]$fallback) {
    $m = Load-Messages
    if ($null -ne $m -and $null -ne $m.$name) { return [string]$m.$name }
    return $fallback
}

function Test-BinaryHashes {
    if (!(Test-Path -LiteralPath $ChecksumsPath)) { return $false }
    try {
        $expected = @{}
        foreach ($line in (Get-Content -LiteralPath $ChecksumsPath -Encoding UTF8)) {
            if ($line -match '^([0-9a-fA-F]{64})\s\s(.+\.exe)$') {
                $expected[$matches[2]] = $matches[1].ToLowerInvariant()
            }
        }
        if ($expected.Count -eq 0) { return $false }
        foreach ($name in $expected.Keys) {
            $path = Join-Path $AppDir $name
            if (!(Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
            $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actual -ne $expected[$name]) { return $false }
        }
        return $true
    } catch { return $false }
}

function Test-GitHubOnline {
    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $async = $client.BeginConnect('github.com', 443, $null, $null)
        if ($async.AsyncWaitHandle.WaitOne(1800, $false) -and $client.Connected) {
            $client.EndConnect($async)
            return $true
        }
    } catch {} finally { try { $client.Close() } catch {} }
    return $false
}

function Ensure-FirewallConfigured {
    if ($TestMode) { return $true }
    if (!(Test-Path -LiteralPath $ConfigPath -PathType Leaf)) { return $false }
    if (!(Test-Path -LiteralPath $FirewallHelperPath -PathType Leaf)) { return $false }

    try {
        $cfg = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $subnet = [string]$cfg.transferSubnet
        $port = [int]$cfg.port
        if ([string]::IsNullOrWhiteSpace($subnet) -or $port -lt 1 -or $port -gt 65535) { return $false }
        $desired = ($AppDir + '|' + $port + '|' + $subnet)
        $marker = ''
        if (Test-Path -LiteralPath $FirewallMarkerPath -PathType Leaf) {
            $marker = (Get-Content -LiteralPath $FirewallMarkerPath -Raw -Encoding UTF8).Trim()
        }
        $x64Rule = Get-NetFirewallRule -DisplayName 'Offline File Transfer' -ErrorAction SilentlyContinue
        $armNeeded = Test-Path -LiteralPath (Join-Path $AppDir 'LocalPhoneTransfer_ARM64.exe') -PathType Leaf
        $armRule = if ($armNeeded) { Get-NetFirewallRule -DisplayName 'Offline File Transfer ARM64' -ErrorAction SilentlyContinue } else { $true }
        if ($marker -eq $desired -and $null -ne $x64Rule -and $null -ne $armRule) { return $true }

        Write-Host (Text 'configuringFirewall' 'Configuring Windows Firewall...') -ForegroundColor Yellow
        $argLine = '-NoProfile -ExecutionPolicy Bypass -File "' + $FirewallHelperPath + '"'
        $p = Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList $argLine -PassThru -Wait
        if ($null -eq $p -or $p.ExitCode -ne 0) { return $false }

        if (!(Test-Path -LiteralPath $FirewallMarkerPath -PathType Leaf)) { return $false }
        $marker = (Get-Content -LiteralPath $FirewallMarkerPath -Raw -Encoding UTF8).Trim()
        return ($marker -eq $desired)
    } catch {
        return $false
    }
}

function Invoke-ReleaseUpdateIfNeeded {
    if ($SkipUpdate) { return $false }
    if (!(Test-Path -LiteralPath $VersionPath)) { return $false }
    if (!(Test-Path -LiteralPath $UpdaterPath)) { return $false }

    $current = (Get-Content -LiteralPath $VersionPath -Raw -Encoding UTF8).Trim()
    if ($current -notmatch '^v[0-9A-Za-z][0-9A-Za-z._-]*$') { return $false }

    Write-Host (Text 'checkingUpdate' 'Checking for updates...')
    if (!(Test-GitHubOnline)) {
        Write-Host (Text 'offlineUpdateSkipped' 'No Internet connection. Update check was skipped.') -ForegroundColor DarkGray
        return $false
    }

    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $headers = @{ 'User-Agent' = 'OfflineFileTransfer' }
        $release = Invoke-RestMethod -UseBasicParsing -Headers $headers -Uri $LatestReleaseApi -TimeoutSec 15
        $latest = [string]$release.tag_name
        if ($latest -notmatch '^v[0-9A-Za-z][0-9A-Za-z._-]*$') { return $false }
        if ($latest -eq $current) {
            Write-Host (Text 'latest' 'The application is up to date.') -ForegroundColor Green
            return $false
        }

        Write-Host (Text 'updating' 'Updating to the latest version...') -ForegroundColor Yellow
        $argLine = '-NoProfile -ExecutionPolicy Bypass -File "' + $UpdaterPath + '" -TargetVersion "' + $latest + '"'
        $p = Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList $argLine -PassThru -Wait
        if ($null -eq $p -or $p.ExitCode -ne 0) {
            Write-Host (Text 'updateFailed' 'Update failed. Starting the current version.') -ForegroundColor Yellow
            return $false
        }

        if (!(Test-BinaryHashes)) {
            Write-Host (Text 'integrityFailed' 'Application integrity check failed after update.') -ForegroundColor Red
            exit 1
        }
        Write-Host (Text 'updated' 'Updated to the latest version.') -ForegroundColor Green

        $newBootstrap = Join-Path $AppDir 'Bootstrap.ps1'
        if ($TestMode) {
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $newBootstrap -TestMode -SkipUpdate
        } else {
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $newBootstrap -SkipUpdate
        }
        exit $LASTEXITCODE
    } catch {
        Write-Host (Text 'updateSkipped' 'Update check could not be completed. Starting the current version.') -ForegroundColor DarkGray
        return $false
    }
}

Write-Host ''
Write-Host (Text 'preparing' 'Preparing file transfer...') -ForegroundColor Cyan

if (!(Test-BinaryHashes)) {
    Write-Host (Text 'integrityFailed' 'Application integrity check failed. Download the application again.') -ForegroundColor Red
    exit 1
}

[void](Invoke-ReleaseUpdateIfNeeded)

if (!(Ensure-FirewallConfigured)) {
    Write-Host (Text 'firewallFailed' 'Windows Firewall could not be configured.') -ForegroundColor Red
    exit 1
}

$startScript = Join-Path $AppDir 'Start-Transfer.ps1'
if (!(Test-Path -LiteralPath $startScript)) {
    Write-Host (Text 'startScriptMissing' 'Start-Transfer.ps1 was not found.') -ForegroundColor Red
    exit 1
}

if ($TestMode) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $startScript -TestMode
} else {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $startScript
}
exit $LASTEXITCODE
