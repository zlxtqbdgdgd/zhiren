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

[Dirs]
; 数据放在所有用户可写的 ProgramData 下，避免装在 Program Files 时无写权限
Name: "{commonappdata}\zhiren\data"; Permissions: users-modify

[Files]
Source: "..\dist\{#MyAppExe}"; DestDir: "{app}"; Flags: ignoreversion
; 绿色浏览器随包打入：若构建时 dist\browser\chrome.exe 存在，则一并安装
#if FileExists(AddBackslash(SourcePath) + "..\dist\browser\chrome.exe")
  #define HasBrowser
  Source: "..\dist\browser\*"; DestDir: "{app}\browser"; Flags: recursesubdirs ignoreversion
#endif

[Icons]
#ifdef HasBrowser
Name: "{group}\知人"; Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"" -browser ""{app}\browser\chrome.exe"""; WorkingDir: "{app}"
Name: "{autodesktop}\知人"; Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"" -browser ""{app}\browser\chrome.exe"""; WorkingDir: "{app}"
#else
Name: "{group}\知人"; Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"""; WorkingDir: "{app}"
Name: "{autodesktop}\知人"; Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"""; WorkingDir: "{app}"
#endif

[Run]
#ifdef HasBrowser
Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"" -browser ""{app}\browser\chrome.exe"""; Description: "立即启动知人"; Flags: nowait postinstall skipifsilent
#else
Filename: "{app}\{#MyAppExe}"; Parameters: "-data ""{commonappdata}\zhiren\data\zhiren.json"""; Description: "立即启动知人"; Flags: nowait postinstall skipifsilent
#endif
