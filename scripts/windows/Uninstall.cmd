@echo off
setlocal
set "SCRIPTFILE=%~f0"
set "PS_SCRIPT=%~dp0Uninstall.ps1"

if not exist "%PS_SCRIPT%" (
  echo Uninstall.ps1 was not found.
  pause
  exit /b 1
)

net session >nul 2>&1
if %errorlevel% neq 0 (
  echo Requesting administrator privileges...
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PS_SCRIPT%"
set "RC=%errorlevel%"
if not "%RC%"=="0" pause
exit /b %RC%
