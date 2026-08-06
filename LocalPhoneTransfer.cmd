@echo off
setlocal
set "APPDIR=%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%APPDIR%Bootstrap.ps1"
if %errorlevel% neq 0 (
  echo.
  echo Local Phone Transfer failed to start.
  pause
)
