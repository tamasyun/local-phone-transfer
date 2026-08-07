param(
    [Parameter(Mandatory=$true)]
    [string]$TargetVersion
)

$ErrorActionPreference = 'Stop'
$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoApi = 'https://api.github.com/repos/tamasyun/local-phone-transfer'

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Test-PackageHashes([string]$dir) {
    $sumPath = Join-Path $dir 'SHA256SUMS.txt'
    if (!(Test-Path -LiteralPath $sumPath)) { return $false }
    foreach ($line in (Get-Content -LiteralPath $sumPath -Encoding UTF8)) {
        if ($line -match '^([0-9a-fA-F]{64})\s\s(.+\.exe)$') {
            $path = Join-Path $dir $matches[2]
            if (!(Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
            $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actual -ne $matches[1].ToLowerInvariant()) { return $false }
        }
    }
    return $true
}

if (!(Test-IsAdministrator)) {
    Write-Error 'Administrator privileges are required for application updates.'
    exit 10
}

if ($TargetVersion -notmatch '^v[0-9A-Za-z][0-9A-Za-z._-]*$') {
    Write-Error 'Invalid release version.'
    exit 11
}

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ('OfflineFileTransfer-update-' + [Guid]::NewGuid().ToString('N'))
$downloadDir = Join-Path $tempRoot 'download'
$extractDir = Join-Path $tempRoot 'extract'
$backupDir = Join-Path $tempRoot 'backup'
New-Item -ItemType Directory -Path $downloadDir,$extractDir,$backupDir -Force | Out-Null

try {
    $headers = @{ 'User-Agent' = 'OfflineFileTransfer-Updater' }
    $release = Invoke-RestMethod -UseBasicParsing -Headers $headers -Uri ($RepoApi + '/releases/tags/' + $TargetVersion) -TimeoutSec 20
    if ($release.draft -eq $true) { throw 'Draft releases cannot be installed.' }

    $zipName = 'OfflineFileTransfer-' + $TargetVersion + '.zip'
    $sumName = $zipName + '.sha256'
    $zipAsset = $release.assets | Where-Object { $_.name -eq $zipName } | Select-Object -First 1
    $sumAsset = $release.assets | Where-Object { $_.name -eq $sumName } | Select-Object -First 1
    if ($null -eq $zipAsset -or $null -eq $sumAsset) { throw 'Required release assets were not found.' }

    $zipPath = Join-Path $downloadDir $zipName
    $sumPath = Join-Path $downloadDir $sumName
    Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri $zipAsset.browser_download_url -OutFile $zipPath -TimeoutSec 120
    Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri $sumAsset.browser_download_url -OutFile $sumPath -TimeoutSec 30

    $sumText = (Get-Content -LiteralPath $sumPath -Raw -Encoding UTF8).Trim()
    if ($sumText -notmatch '^([0-9a-fA-F]{64})\s+') { throw 'Release checksum file is invalid.' }
    $expectedZip = $matches[1].ToLowerInvariant()
    $actualZip = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualZip -ne $expectedZip) { throw 'Release ZIP checksum verification failed.' }

    Expand-Archive -LiteralPath $zipPath -DestinationPath $extractDir -Force
    $source = Join-Path $extractDir 'OfflineFileTransfer\app'
    if (!(Test-Path -LiteralPath (Join-Path $source 'Bootstrap.ps1'))) { throw 'Release package layout is invalid.' }
    if (!(Test-PackageHashes $source)) { throw 'Release executable verification failed.' }

    # Back up managed application files only. User data and configuration are preserved separately.
    robocopy $AppDir $backupDir /E /COPY:DAT /R:1 /W:1 /XD received-files shared-files logs /XF transfer-config.json .installed /NFL /NDL /NJH /NJS /NP | Out-Null
    if ($LASTEXITCODE -ge 8) { throw 'Could not create update backup.' }

    robocopy $source $AppDir /E /COPY:DAT /R:2 /W:1 /XF transfer-config.json /NFL /NDL /NJH /NJS /NP | Out-Null
    if ($LASTEXITCODE -ge 8) { throw 'Could not install update files.' }

    if (!(Test-PackageHashes $AppDir)) { throw 'Installed executable verification failed.' }
    Set-Content -LiteralPath (Join-Path $AppDir 'VERSION.txt') -Value $TargetVersion -Encoding ASCII
    exit 0
}
catch {
    $message = $_.Exception.Message
    try {
        if (Test-Path -LiteralPath (Join-Path $backupDir 'Bootstrap.ps1')) {
            robocopy $backupDir $AppDir /E /COPY:DAT /R:2 /W:1 /NFL /NDL /NJH /NJS /NP | Out-Null
        }
    } catch {}
    Write-Error $message
    exit 1
}
finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
