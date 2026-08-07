param(
    [Parameter(Mandatory = $true)]
    [string]$InstallDir
)

$ErrorActionPreference = 'Stop'

$configPath = Join-Path $InstallDir 'transfer-config.json'
if (!(Test-Path -LiteralPath $configPath)) {
    throw 'transfer-config.json was not found.'
}

$config = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8 | ConvertFrom-Json
$subnet = [string]$config.transferSubnet
$port = [int]$config.port
if ([string]::IsNullOrWhiteSpace($subnet)) { throw 'transferSubnet is missing.' }
if ($port -lt 1 -or $port -gt 65535) { throw 'port is invalid.' }

$x64 = Join-Path $InstallDir 'LocalPhoneTransfer.exe'
$arm64 = Join-Path $InstallDir 'LocalPhoneTransfer_ARM64.exe'
if (!(Test-Path -LiteralPath $x64)) { throw 'LocalPhoneTransfer.exe was not found.' }

$dataDirs = @(
    (Join-Path $InstallDir 'received-files'),
    (Join-Path $InstallDir 'shared-files'),
    (Join-Path $InstallDir 'logs')
)

foreach ($path in $dataDirs) {
    New-Item -ItemType Directory -Path $path -Force | Out-Null
    & icacls.exe $path /inheritance:e /grant '*S-1-5-32-545:(OI)(CI)M' /T /C | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw ('Failed to set data directory permissions: ' + $path)
    }
}

$rules = @(
    @{ Name = 'Offline File Transfer'; Program = $x64 },
    @{ Name = 'Offline File Transfer ARM64'; Program = $arm64 }
)

foreach ($rule in $rules) {
    Get-NetFirewallRule -DisplayName $rule.Name -ErrorAction SilentlyContinue |
        Remove-NetFirewallRule -ErrorAction SilentlyContinue

    if (Test-Path -LiteralPath $rule.Program) {
        New-NetFirewallRule `
            -DisplayName $rule.Name `
            -Direction Inbound `
            -Action Allow `
            -Protocol TCP `
            -LocalPort $port `
            -Program $rule.Program `
            -RemoteAddress $subnet `
            -Profile Any | Out-Null
    }
}
