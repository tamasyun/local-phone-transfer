@echo off
setlocal
set "APPDIR=%~dp0"
if not exist "%APPDIR%.installed" (
  echo.
  echo This is an internal test launcher.
  echo Install the application with Setup.cmd first.
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
