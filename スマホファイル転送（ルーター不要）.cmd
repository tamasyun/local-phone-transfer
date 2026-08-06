@echo off
chcp 65001 >nul
setlocal
set "APPDIR=%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%APPDIR%Start-RouterlessTransfer.ps1"
if %errorlevel% neq 0 (
  echo.
  echo ルーターなし転送の起動に失敗しました。
  echo 上に表示されたエラー内容を確認してください。
  pause
)
