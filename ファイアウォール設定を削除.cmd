@echo off
chcp 65001 >nul
set "SCRIPTFILE=%~f0"
net session >nul 2>&1
if %errorlevel% neq 0 (
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)
powershell.exe -NoProfile -Command "Get-NetFirewallRule -DisplayName 'スマホファイル転送（ローカルネットワーク）','スマホファイル転送 ARM64（ローカルネットワーク）' -ErrorAction SilentlyContinue | Remove-NetFirewallRule"
echo ファイアウォール設定を削除しました。
pause
