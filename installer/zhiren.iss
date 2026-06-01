; 知人 · 干部信息管理系统 — Inno Setup 安装脚本
#define MyAppName "知人 干部信息管理系统"
#ifndef MyAppVersion
  #define MyAppVersion "0.0.0-dev"
#endif
#define MyAppExe "zhiren.exe"

[Setup]
AppId={{D7B9E6A2-4C3F-4E9A-9C1B-7A2F3E5D8C10}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher=zhiren
DefaultDirName={autopf}\zhiren
DefaultGroupName=知人
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=zhiren-setup-{#MyAppVersion}
Compression=lzma2
SolidCompression=yes
; 支持 Windows 7 及以上（6.1 = Win7）
MinVersion=6.1
WizardStyle=modern

[Languages]
Name: "default"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "..\dist\{#MyAppExe}"; DestDir: "{app}"; Flags: ignoreversion
; TODO(v1): 绿色免安装 Chromium 浏览器随包打入 {app}\browser\，
;           并把下方桌面快捷方式改为用它打开 http://localhost:8080

[Icons]
Name: "{group}\知人"; Filename: "{app}\{#MyAppExe}"
Name: "{autodesktop}\知人"; Filename: "{app}\{#MyAppExe}"

[Run]
Filename: "{app}\{#MyAppExe}"; Description: "立即启动知人"; Flags: nowait postinstall skipifsilent
