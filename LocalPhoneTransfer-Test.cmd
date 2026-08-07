@echo off
setlocal
set "APPDIR=%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%APPDIR%Bootstrap.ps1" -TestMode
if %errorlevel% neq 0 (
  echo.
  echo Test mode failed to start.
  echo Check the error message above.
  pause
)
