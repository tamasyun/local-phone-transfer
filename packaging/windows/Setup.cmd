@echo off
setlocal
set "PACKAGE_DIR=%~dp0"
set "SOURCE_APP=%~dp0app"
set "INSTALL_DIR=%ProgramData%\OfflineFileTransfer"
set "SCRIPTFILE=%~f0"

net session >nul 2>&1
if %errorlevel% neq 0 (
  echo Requesting administrator privileges...
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)

if not exist "%SOURCE_APP%\Launch.cmd" goto :missing
if not exist "%SOURCE_APP%\Bootstrap.ps1" goto :missing
if not exist "%SOURCE_APP%\Start-Transfer.ps1" goto :missing
if not exist "%SOURCE_APP%\transfer-config.json" goto :missing
if not exist "%SOURCE_APP%\SHA256SUMS.txt" goto :missing

if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%" >nul 2>&1
robocopy "%SOURCE_APP%" "%INSTALL_DIR%" /E /COPY:DAT /DCOPY:DAT /R:2 /W:1 /NFL /NDL /NJH /NJS /NP >nul
if %errorlevel% geq 8 goto :copyfail

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference='Stop';" ^
  "$dir=$env:INSTALL_DIR;" ^
  "$cfg=Get-Content -LiteralPath (Join-Path $dir 'transfer-config.json') -Raw -Encoding UTF8 | ConvertFrom-Json;" ^
  "$x64=Join-Path $dir 'LocalPhoneTransfer.exe'; $arm=Join-Path $dir 'LocalPhoneTransfer_ARM64.exe';" ^
  "if(!(Test-Path -LiteralPath $x64)){throw 'LocalPhoneTransfer.exe not found.'};" ^
  "$subnet=[string]$cfg.transferSubnet; if([string]::IsNullOrWhiteSpace($subnet)){throw 'transferSubnet is missing.'};" ^
  "$name='Offline File Transfer'; $nameArm='Offline File Transfer ARM64';" ^
  "Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "Get-NetFirewallRule -DisplayName $nameArm -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $x64 -RemoteAddress $subnet -Profile Any | Out-Null;" ^
  "if(Test-Path -LiteralPath $arm){New-NetFirewallRule -DisplayName $nameArm -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $arm -RemoteAddress $subnet -Profile Any | Out-Null};" ^
  "$shortcutName=[string]$cfg.shortcutName; if([string]::IsNullOrWhiteSpace($shortcutName)){$shortcutName='Offline File Transfer'};" ^
  "$desktop=[Environment]::GetFolderPath('Desktop'); $shortcut=Join-Path $desktop ($shortcutName+'.lnk');" ^
  "$ws=New-Object -ComObject WScript.Shell; $sc=$ws.CreateShortcut($shortcut);" ^
  "$sc.TargetPath=Join-Path $dir 'Launch.cmd'; $sc.WorkingDirectory=$dir; $sc.Description='Offline file transfer between PC and smartphone'; $sc.Save();" ^
  "Set-Content -LiteralPath (Join-Path $dir '.installed') -Value 'installed' -Encoding ASCII;"
if %errorlevel% neq 0 goto :setupfail

echo.
echo ================================================
echo Setup completed.
echo ================================================
echo.
echo Start the application from the desktop shortcut.
echo.
pause
exit /b 0

:missing
echo.
echo Setup files are incomplete.
echo Download and extract the official release package again.
pause
exit /b 1

:copyfail
echo.
echo Failed to install application files.
pause
exit /b 1

:setupfail
echo.
echo Failed to configure Windows Firewall or the desktop shortcut.
pause
exit /b 1
