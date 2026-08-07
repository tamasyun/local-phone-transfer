param(
    [switch]$TestMode
)

$ErrorActionPreference = 'SilentlyContinue'
$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$LegacyLauncher = Join-Path $AppDir 'LocalPhoneTransfer.cmd'

if (Test-Path -LiteralPath $LegacyLauncher) {
    & cmd.exe /c ('"' + $LegacyLauncher + '"')
    exit $LASTEXITCODE
}

Start-Process 'https://github.com/tamasyun/local-phone-transfer/releases'
exit 0
