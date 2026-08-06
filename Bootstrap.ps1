$ErrorActionPreference = 'SilentlyContinue'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$MessagesPath = Join-Path $AppDir 'messages.json'

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

Write-Host ''
Write-Host (Text 'preparing' 'Preparing file transfer...') -ForegroundColor Cyan

$git = Get-Command git.exe -ErrorAction SilentlyContinue
$gitDir = Join-Path $AppDir '.git'
if ($null -ne $git -and (Test-Path -LiteralPath $gitDir)) {
    $branch = (& git.exe -C $AppDir rev-parse --abbrev-ref HEAD 2>$null)
    if ($LASTEXITCODE -eq 0 -and $branch -eq 'main') {
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
                        if ($LASTEXITCODE -eq 0) {
                            Write-Host (Text 'updated' 'Updated to the latest version.') -ForegroundColor Green
                        } else {
                            Write-Host (Text 'updateFailed' 'Update failed. Starting the current version.') -ForegroundColor Yellow
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

& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $startScript
exit $LASTEXITCODE
