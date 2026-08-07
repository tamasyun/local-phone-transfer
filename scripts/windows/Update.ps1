param(
    [Parameter(Mandatory=$true)]
    [string]$TargetVersion
)

$ErrorActionPreference = 'Stop'
$RepoApi = 'https://api.github.com/repos/tamasyun/local-phone-transfer'

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object System.Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
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
New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null

try {
    $headers = @{ 'User-Agent' = 'OfflineFileTransfer-Updater' }
    $release = Invoke-RestMethod -UseBasicParsing -Headers $headers -Uri ($RepoApi + '/releases/tags/' + $TargetVersion) -TimeoutSec 20
    if ($release.draft -eq $true) { throw 'Draft releases cannot be installed.' }

    $installerName = 'OfflineFileTransfer-' + $TargetVersion + '-Setup.exe'
    $sumName = $installerName + '.sha256'
    $installerAsset = $release.assets | Where-Object { $_.name -eq $installerName } | Select-Object -First 1
    $sumAsset = $release.assets | Where-Object { $_.name -eq $sumName } | Select-Object -First 1
    if ($null -eq $installerAsset -or $null -eq $sumAsset) { throw 'Required installer assets were not found.' }

    $installerPath = Join-Path $tempRoot $installerName
    $sumPath = Join-Path $tempRoot $sumName
    Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri $installerAsset.browser_download_url -OutFile $installerPath -TimeoutSec 180
    Invoke-WebRequest -UseBasicParsing -Headers $headers -Uri $sumAsset.browser_download_url -OutFile $sumPath -TimeoutSec 30

    $sumText = (Get-Content -LiteralPath $sumPath -Raw -Encoding UTF8).Trim()
    if ($sumText -notmatch '^([0-9a-fA-F]{64})\s+') { throw 'Release checksum file is invalid.' }
    $expected = $matches[1].ToLowerInvariant()
    $actual = (Get-FileHash -LiteralPath $installerPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw 'Installer checksum verification failed.' }

    $arguments = '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS'
    $process = Start-Process -FilePath $installerPath -ArgumentList $arguments -PassThru -Wait
    if ($null -eq $process -or $process.ExitCode -ne 0) {
        throw ('Installer exited with code ' + $(if ($null -eq $process) { 'unknown' } else { $process.ExitCode }))
    }

    exit 0
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
