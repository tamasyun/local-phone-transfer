@echo off
setlocal
set "SCRIPTFILE=%~f0"
net session >nul 2>&1
if %errorlevel% neq 0 (
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)
powershell.exe -NoProfile -Command "Get-NetFirewallRule -DisplayName 'Local Phone Transfer','Local Phone Transfer ARM64' -ErrorAction SilentlyContinue | Remove-NetFirewallRule"
echo Firewall rules removed.
pause
