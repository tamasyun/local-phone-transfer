$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ConfigPath = Join-Path $AppDir 'transfer-config.json'
$MarkerPath = Join-Path $AppDir '.firewall-configured'

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (!$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Administrator privileges are required to configure Windows Firewall.'
}

if (!(Test-Path -LiteralPath $ConfigPath -PathType Leaf)) {
    throw 'transfer-config.json was not found.'
}

$config = Get-Content -LiteralPath $ConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
$subnet = [string]$config.transferSubnet
$port = [int]$config.port
if ([string]::IsNullOrWhiteSpace($subnet)) { throw 'transferSubnet is missing.' }
if ($port -lt 1 -or $port -gt 65535) { throw 'port is invalid.' }

$x64 = Join-Path $AppDir 'LocalPhoneTransfer.exe'
$arm64 = Join-Path $AppDir 'LocalPhoneTransfer_ARM64.exe'
if (!(Test-Path -LiteralPath $x64 -PathType Leaf)) { throw 'LocalPhoneTransfer.exe was not found.' }

$rules = @('Offline File Transfer', 'Offline File Transfer ARM64')
foreach ($name in $rules) {
    Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue |
        Remove-NetFirewallRule -ErrorAction SilentlyContinue
}

New-NetFirewallRule -DisplayName 'Offline File Transfer' -Direction Inbound -Action Allow -Protocol TCP -LocalPort $port -Program $x64 -RemoteAddress $subnet -Profile Any | Out-Null
if (Test-Path -LiteralPath $arm64 -PathType Leaf) {
    New-NetFirewallRule -DisplayName 'Offline File Transfer ARM64' -Direction Inbound -Action Allow -Protocol TCP -LocalPort $port -Program $arm64 -RemoteAddress $subnet -Profile Any | Out-Null
}

$state = ($AppDir + '|' + $port + '|' + $subnet)
Set-Content -LiteralPath $MarkerPath -Value $state -Encoding UTF8
