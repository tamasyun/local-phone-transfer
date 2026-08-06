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
  "$cfgPath=Join-Path $dir 'transfer-config.json';" ^
  "$cfg=Get-Content -LiteralPath $cfgPath -Raw -Encoding UTF8 | ConvertFrom-Json;" ^
  "$exe=Join-Path $dir 'スマホファイル転送.exe';" ^
  "$exeArm=Join-Path $dir 'スマホファイル転送_ARM64.exe';" ^
  "if(!(Test-Path $exe)){throw 'スマホファイル転送.exe が見つかりません'};" ^
  "$name='スマホファイル転送（ローカルネットワーク）';" ^
  "$nameArm='スマホファイル転送 ARM64（ローカルネットワーク）';" ^
  "Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "Get-NetFirewallRule -DisplayName $nameArm -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue;" ^
  "New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exe -RemoteAddress LocalSubnet -Profile Any | Out-Null;" ^
  "if(Test-Path $exeArm){New-NetFirewallRule -DisplayName $nameArm -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$cfg.port) -Program $exeArm -RemoteAddress LocalSubnet -Profile Any | Out-Null};" ^
  "$launcher=Join-Path $dir 'スマホファイル転送.cmd';" ^
  "$desktop=[Environment]::GetFolderPath('Desktop');" ^
  "$shortcut=Join-Path $desktop 'スマホファイル転送.lnk';" ^
  "$ws=New-Object -ComObject WScript.Shell;" ^
  "$sc=$ws.CreateShortcut($shortcut);" ^
  "$sc.TargetPath=$launcher; $sc.WorkingDirectory=$dir; $sc.Description='iPhone / Android と共用PCのファイル転送'; $sc.Save();" ^
  "Write-Host ''; Write-Host 'ファイアウォール設定とデスクトップショートカットを作成しました。' -ForegroundColor Green;"

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
echo デスクトップの「スマホファイル転送」から起動できます。
echo.
echo transfer-config.json の wifiPassword が CHANGE-ME のままなら、
echo 「設定を編集.cmd」から8文字以上のパスワードに変更してください。
echo.
pause
