@echo off
setlocal
set "APPDIR=%~dp0"
set "SCRIPTFILE=%~f0"

net session >nul 2>&1
if %errorlevel% neq 0 (
  echo Requesting administrator privileges...
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference='Stop';" ^
  "$dir=$env:APPDIR;" ^
  "$cfgPath=Join-Path $dir 'transfer-config.json';" ^
  "$cfg=Get-Content -LiteralPath $cfgPath -Raw -Encoding UTF8 | ConvertFrom-Json;" ^
  "$allExe=Get-ChildItem -LiteralPath $dir -Filter '*.exe' -File;" ^
  "$exe=$allExe | Where-Object { $_.Name -notlike '*_ARM64.exe' } | Select-Object -First 1;" ^
  "$exeArm=$allExe | Where-Object { $_.Name -like '*_ARM64.exe' } | Select-Object -First 1;" ^
  "if($null -eq $exe){throw 'Transfer executable not found.'};" ^
  "$name='Local Phone Transfer';" ^
  "$nameArm='Local Phone Transfer ARM64';" ^
  "Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "Get-NetFirewallRule -DisplayName $nameArm -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exe.FullName -RemoteAddress LocalSubnet -Profile Any | Out-Null;" ^
  "if($null -ne $exeArm){New-NetFirewallRule -DisplayName $nameArm -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exeArm.FullName -RemoteAddress LocalSubnet -Profile Any | Out-Null};" ^
  "$launcher=Join-Path $dir 'LocalPhoneTransfer.cmd';" ^
  "if(!(Test-Path -LiteralPath $launcher)){throw 'LocalPhoneTransfer.cmd not found.'};" ^
  "$desktop=[Environment]::GetFolderPath('Desktop');" ^
  "$shortcut=Join-Path $desktop 'Local Phone Transfer.lnk';" ^
  "$ws=New-Object -ComObject WScript.Shell;" ^
  "$sc=$ws.CreateShortcut($shortcut);" ^
  "$sc.TargetPath=$launcher; $sc.WorkingDirectory=$dir; $sc.Description='Local file transfer for iPhone and Android'; $sc.Save();" ^
  "Write-Host ''; Write-Host 'Firewall rules and desktop shortcut created.' -ForegroundColor Green;"

if %errorlevel% neq 0 (
  echo.
  echo Setup failed.
  pause
  exit /b 1
)

echo.
echo ================================================
echo Setup completed.
echo ================================================
echo.
echo Start from the "Local Phone Transfer" desktop shortcut.
echo If wifiPassword is still CHANGE-ME, edit transfer-config.json.
echo.
pause
