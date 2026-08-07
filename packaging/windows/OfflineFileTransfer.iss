#define MyAppName "Offline File Transfer"
#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif
#ifndef BuildRoot
  #define BuildRoot ".\build\app"
#endif
#ifndef OutputDir
  #define OutputDir ".\output"
#endif

[Setup]
AppId={{D1B5E2DD-9F0C-4B62-B29E-417DB50A2A3E}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher=tamasyun
AppPublisherURL=https://github.com/tamasyun/local-phone-transfer
AppSupportURL=https://github.com/tamasyun/local-phone-transfer/issues
AppUpdatesURL=https://github.com/tamasyun/local-phone-transfer/releases
DefaultDirName={commonappdata}\OfflineFileTransfer
DisableDirPage=yes
DisableProgramGroupPage=yes
PrivilegesRequired=admin
MinVersion=10.0.22000
OutputDir={#OutputDir}
OutputBaseFilename=OfflineFileTransfer-{#MyAppVersion}-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayName=Offline File Transfer
CreateUninstallRegKey=yes
ChangesEnvironment=no
CloseApplications=yes
RestartApplications=no
SetupLogging=yes

[Files]
Source: "{#BuildRoot}\LocalPhoneTransfer.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\LocalPhoneTransfer_ARM64.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\Bootstrap.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\Start-Transfer.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\Update.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\Install-Configure.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\Launch.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\SHA256SUMS.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\VERSION.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#BuildRoot}\transfer-config.json"; DestDir: "{app}"; Flags: onlyifdoesntexist uninsneveruninstall
Source: "{#BuildRoot}\locales\*"; DestDir: "{app}\locales"; Flags: ignoreversion recursesubdirs createallsubdirs

[Dirs]
Name: "{app}\received-files"; Permissions: users-modify; Flags: uninsneveruninstall
Name: "{app}\shared-files"; Permissions: users-modify; Flags: uninsneveruninstall
Name: "{app}\logs"; Permissions: users-modify; Flags: uninsneveruninstall

[Icons]
Name: "{commondesktop}\オフラインファイル転送（PC↔スマホ）"; Filename: "{app}\Launch.cmd"; WorkingDir: "{app}"; Comment: "PCとスマートフォン間のオフラインファイル転送"

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\Install-Configure.ps1"" -InstallDir ""{app}"""; Flags: runhidden waituntilterminated; StatusMsg: "Configuring Offline File Transfer..."

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -Command ""Get-NetFirewallRule -DisplayName 'Offline File Transfer','Offline File Transfer ARM64' -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue"""; Flags: runhidden waituntilterminated

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    SaveStringToFile(ExpandConstant('{app}\.installed'), 'installed'#13#10, False);
  end;
end;
