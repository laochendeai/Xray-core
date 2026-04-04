#define MyAppName "Xray-core"
#define MyAppPublisher "laochendeai"
#define MyAppURL "https://github.com/laochendeai/Xray-core"
#define MyAppExeName "xray.exe"

#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif

#ifndef MySourceDir
  #define MySourceDir "."
#endif

#ifndef MyOutputDir
  #define MyOutputDir "."
#endif

#ifndef MyOutputBaseFilename
  #define MyOutputBaseFilename "Xray-windows-64-setup"
#endif

[Setup]
AppId={{F8831BD1-5B00-41F8-8DB1-12C175B0A6DB}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\Xray-core
DisableProgramGroupPage=yes
LicenseFile={#MySourceDir}\LICENSE
OutputDir={#MyOutputDir}
OutputBaseFilename={#MyOutputBaseFilename}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible and not arm64
ArchitecturesInstallIn64BitMode=x64compatible and not arm64
PrivilegesRequired=admin
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "{#MySourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
