@echo off
chcp 65001 >nul
setlocal
set "APPDIR=%~dp0"
set "SCRIPTFILE=%~f0"

net session >nul 2>&1
if %errorlevel% neq 0 (
  echo 管理者権限を要求します...
  powershell.exe -NoProfile -Command "Start-Process -FilePath $env:SCRIPTFILE -Verb RunAs"
  exit /b
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference='Stop';" ^
  "$dir=$env:APPDIR;" ^
  "$exe=Join-Path $dir 'スマホファイル転送.exe';" ^
  "$cfgPath=Join-Path $dir 'transfer-config.json';" ^
  "if(!(Test-Path $exe)){throw 'スマホファイル転送.exe が見つかりません'};" ^
  "$cfg=Get-Content -LiteralPath $cfgPath -Raw -Encoding UTF8 | ConvertFrom-Json;" ^
  "$name='スマホファイル転送（ローカルネットワーク）';" ^
  "$nameArm='スマホファイル転送 ARM64（ローカルネットワーク）';" ^
  "Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "Get-NetFirewallRule -DisplayName $nameArm -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exe -RemoteAddress LocalSubnet -Profile Any | Out-Null;" ^
  "$exeArm=Join-Path $dir 'スマホファイル転送_ARM64.exe'; if(Test-Path $exeArm){New-NetFirewallRule -DisplayName $nameArm -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exeArm -RemoteAddress LocalSubnet -Profile Any | Out-Null};" ^
  "Write-Host ''; Write-Host 'ファイアウォール設定を追加しました。' -ForegroundColor Green;" ^
  "Write-Host ('TCPポート: ' + $cfg.port); Write-Host ('対象EXE: ' + $exe);"

if %errorlevel% neq 0 (
  echo.
  echo セットアップに失敗しました。
  pause
  exit /b 1
)

echo.
echo ================================================
echo 初期セットアップが完了しました。
echo ================================================
echo.
echo 次に transfer-config.json の wifiSsid と wifiPassword を
echo 転送専用Wi-Fiルーターと同じ値に変更してください。
echo.
echo 設定ファイルを開きます。
start "" notepad.exe "%APPDIR%transfer-config.json"
echo.
pause
