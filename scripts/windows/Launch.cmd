@echo off
setlocal
set "APPDIR=%~dp0"
if not exist "%APPDIR%.installed" (
  echo.
  echo This is an internal application file.
  echo Install Offline File Transfer using the official installer first.
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
