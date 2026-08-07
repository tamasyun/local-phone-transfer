@echo off
setlocal
set "APPDIR=%~dp0app"
if not exist "%APPDIR%\Launch.cmd" (
  echo.
  echo Application files are incomplete.
  echo Download and extract the official release ZIP again.
  echo.
  pause
  exit /b 1
)
call "%APPDIR%\Launch.cmd"
