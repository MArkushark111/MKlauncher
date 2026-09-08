[Setup]
AppName=MKLauncher
AppVersion=1.0.4
AppVerName=MKLauncher
AppPublisher=MKGames
DefaultDirName={autopf}\MKLauncher
DefaultGroupName=MKLauncher
OutputBaseFilename=MKLauncher-Setup
SetupIconFile=C:\Users\Mark\Music\mkgames_\mkgameslogo.ico
UninstallIconFile=C:\Users\Mark\Music\mkgames_\mkgameslogo.ico
Compression=lzma2
SolidCompression=yes
OutputDir=C:\Users\Mark\Music\mkgames_\installer\output
WizardStyle=classic
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

[Files]
Source: "C:\Users\Mark\Music\mkgames_\client\build\MKLauncher.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\*.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\platforms\*"; DestDir: "{app}\platforms"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\imageformats\*"; DestDir: "{app}\imageformats"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\generic\*"; DestDir: "{app}\generic"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\iconengines\*"; DestDir: "{app}\iconengines"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\networkinformation\*"; DestDir: "{app}\networkinformation"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\sqldrivers\*"; DestDir: "{app}\sqldrivers"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\styles\*"; DestDir: "{app}\styles"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\tls\*"; DestDir: "{app}\tls"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\translations\*"; DestDir: "{app}\translations"; Flags: ignoreversion

[Icons]
Name: "{group}\MKLauncher"; Filename: "{app}\MKLauncher.exe"; IconFilename: "{app}\MKLauncher.exe"
Name: "{group}\Uninstall MKLauncher"; Filename: "{uninstallexe}"
Name: "{autodesktop}\MKLauncher"; Filename: "{app}\MKLauncher.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Run]
Filename: "{app}\MKLauncher.exe"; Description: "Launch MKLauncher now"; Flags: nowait postinstall skipifsilent
