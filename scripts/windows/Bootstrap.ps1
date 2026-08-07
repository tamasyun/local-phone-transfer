param(
    [switch]$TestMode
)

$ErrorActionPreference = 'SilentlyContinue'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$ChecksumsPath = Join-Path $AppDir 'SHA256SUMS.txt'
$ExpectedOrigin = 'https://github.com/tamasyun/local-phone-transfer.git'

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
            if ($line -match '^([0-9a-fA-F]{64})\s\s(.+)$') {
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

Write-Host ''
Write-Host (Text 'preparing' 'Preparing file transfer...') -ForegroundColor Cyan

if (!(Test-BinaryHashes)) {
    Write-Host (Text 'integrityFailed' 'Application integrity check failed. Reinstall the application.') -ForegroundColor Red
    exit 1
}

# Development clones can still fast-forward from the official repository.
# Installed Release packages do not contain .git and therefore skip this block.
$git = Get-Command git.exe -ErrorAction SilentlyContinue
$gitDir = Join-Path $AppDir '.git'
if ($null -ne $git -and (Test-Path -LiteralPath $gitDir)) {
    $branch = (& git.exe -C $AppDir rev-parse --abbrev-ref HEAD 2>$null)
    $origin = (& git.exe -C $AppDir config --get remote.origin.url 2>$null)
    if ($LASTEXITCODE -eq 0 -and $branch -eq 'main' -and $origin -eq $ExpectedOrigin) {
        Write-Host (Text 'checkingUpdate' 'Checking for updates...')
        $online = $false
        $client = New-Object System.Net.Sockets.TcpClient
        try {
            $async = $client.BeginConnect('github.com', 443, $null, $null)
            if ($async.AsyncWaitHandle.WaitOne(1800, $false) -and $client.Connected) {
                $client.EndConnect($async)
                $online = $true
            }
        } catch {} finally { try { $client.Close() } catch {} }

        if ($online) {
            $env:GIT_TERMINAL_PROMPT = '0'
            & git.exe -C $AppDir fetch origin main --quiet 2>$null
            if ($LASTEXITCODE -eq 0) {
                $local = (& git.exe -C $AppDir rev-parse HEAD 2>$null)
                $remote = (& git.exe -C $AppDir rev-parse origin/main 2>$null)
                if ($local -ne $remote) {
                    $changes = (& git.exe -C $AppDir status --porcelain --untracked-files=no 2>$null)
                    if ([string]::IsNullOrWhiteSpace(($changes -join ''))) {
                        Write-Host (Text 'updating' 'Updating...') -ForegroundColor Yellow
                        & git.exe -C $AppDir merge --ff-only origin/main --quiet 2>$null
                        if ($LASTEXITCODE -eq 0 -and (Test-BinaryHashes)) {
                            Write-Host (Text 'updated' 'Updated to the latest version.') -ForegroundColor Green
                        } else {
                            Write-Host (Text 'integrityRollback' 'The update did not pass integrity verification. Restoring the previous version.') -ForegroundColor Red
                            & git.exe -C $AppDir reset --hard $local --quiet 2>$null
                            if (!(Test-BinaryHashes)) { exit 1 }
                        }
                    } else {
                        Write-Host (Text 'localChanges' 'Local changes were found. Automatic update was skipped.') -ForegroundColor Yellow
                    }
                } else {
                    Write-Host (Text 'latest' 'The application is up to date.') -ForegroundColor Green
                }
            } else {
                Write-Host (Text 'updateSkipped' 'Update check was skipped. Starting the current version.') -ForegroundColor Yellow
            }
        } else {
            Write-Host (Text 'offlineUpdateSkipped' 'No Internet connection. Update check was skipped.') -ForegroundColor DarkGray
        }
    }
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
