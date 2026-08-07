@echo off
setlocal
set "APPDIR=%~dp0"
if not exist "%APPDIR%Bootstrap.ps1" (
  echo.
  echo Bootstrap.ps1 was not found.
  echo.
  pause
  exit /b 1
)
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%APPDIR%Bootstrap.ps1" -TestMode
if %errorlevel% neq 0 (
  echo.
  echo Test mode failed to start.
  pause
)
