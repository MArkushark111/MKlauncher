[Setup]
AppName=MKLauncher
AppVersion=1.0.0
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
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\MKLauncher.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\*.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\platforms\*"; DestDir: "{app}\platforms"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\imageformats\*"; DestDir: "{app}\imageformats"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\generic\*"; DestDir: "{app}\generic"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\iconengines\*"; DestDir: "{app}\iconengines"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\networkinformation\*"; DestDir: "{app}\networkinformation"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\sqldrivers\*"; DestDir: "{app}\sqldrivers"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\styles\*"; DestDir: "{app}\styles"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\tls\*"; DestDir: "{app}\tls"; Flags: ignoreversion
Source: "C:\Users\Mark\Music\mkgames_\client\build\Desktop_Qt_6_11_2_MinGW_64_bit_Debug\translations\*"; DestDir: "{app}\translations"; Flags: ignoreversion

[Icons]
Name: "{group}\MKLauncher"; Filename: "{app}\MKLauncher.exe"; IconFilename: "{app}\MKLauncher.exe"
Name: "{group}\Uninstall MKLauncher"; Filename: "{uninstallexe}"
Name: "{autodesktop}\MKLauncher"; Filename: "{app}\MKLauncher.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Run]
Filename: "{app}\MKLauncher.exe"; Description: "Launch MKLauncher now"; Flags: nowait postinstall skipifsilent

[Code]
var
  ResultCode: Integer;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    Exec(ExpandConstant('{app}\MKLauncher.exe'), '', '', SW_SHOW, ewNoWait, ResultCode);
  end;
end;
