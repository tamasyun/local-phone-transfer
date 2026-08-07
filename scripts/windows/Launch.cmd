@echo off
setlocal
set "APPDIR=%~dp0"
if not exist "%APPDIR%Bootstrap.ps1" (
  echo.
  echo Application files are incomplete.
  echo Download and extract the official release ZIP again.
  echo.
  pause
  exit /b 1
)
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%APPDIR%Bootstrap.ps1"
if %errorlevel% neq 0 (
  echo.
  echo Offline File Transfer failed to start.
  pause
)
