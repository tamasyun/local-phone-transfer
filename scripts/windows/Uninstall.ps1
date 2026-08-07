$ErrorActionPreference = 'Stop'

$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ReceivedDir = Join-Path $AppDir 'received-files'

try {
    Get-Process -Name 'LocalPhoneTransfer','LocalPhoneTransfer_ARM64' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

    Get-NetFirewallRule -DisplayName 'Offline File Transfer','Offline File Transfer ARM64' -ErrorAction SilentlyContinue |
        Remove-NetFirewallRule -ErrorAction SilentlyContinue

    $desktop = [Environment]::GetFolderPath([Environment+SpecialFolder]::CommonDesktopDirectory)
    if (-not [string]::IsNullOrWhiteSpace($desktop)) {
        Remove-Item -LiteralPath (Join-Path $desktop 'Offline File Transfer.cmd') -Force -ErrorAction SilentlyContinue
    }

    if ((Test-Path -LiteralPath $ReceivedDir -PathType Container) -and
        (Get-ChildItem -LiteralPath $ReceivedDir -Force -ErrorAction SilentlyContinue | Select-Object -First 1)) {
        Write-Host ''
        $answer = Read-Host 'Keep received files in your Downloads folder? [Y/n]'
        if ([string]::IsNullOrWhiteSpace($answer) -or $answer -match '^[Yy]') {
            $downloads = Join-Path $env:USERPROFILE 'Downloads'
            if (!(Test-Path -LiteralPath $downloads -PathType Container)) {
                New-Item -ItemType Directory -Path $downloads -Force | Out-Null
            }
            $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
            $backup = Join-Path $downloads ('OfflineFileTransfer-received-' + $stamp)
            Move-Item -LiteralPath $ReceivedDir -Destination $backup
            Write-Host ('Received files were moved to: ' + $backup)
        }
    }

    $parent = Split-Path -Parent $AppDir
    $leaf = Split-Path -Leaf $AppDir
    $cleanup = "Start-Sleep -Milliseconds 800; Remove-Item -LiteralPath '" + $AppDir.Replace("'", "''") + "' -Recurse -Force -ErrorAction SilentlyContinue"
    Start-Process -FilePath 'powershell.exe' -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-Command',$cleanup -WindowStyle Hidden

    Write-Host ''
    Write-Host 'Offline File Transfer was uninstalled.' -ForegroundColor Green
    Write-Host 'This window can be closed.'
}
catch {
    Write-Host ''
    Write-Host ('Uninstall failed: ' + $_.Exception.Message) -ForegroundColor Red
    exit 1
}
