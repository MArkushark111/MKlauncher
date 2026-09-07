@echo off
echo ========================================
echo   MKLauncher Windows Installer Builder
echo ========================================
echo.

set QT_DIR=C:\Qt\6.11.2\mingw_64
set MINGW_DIR=C:\Qt\Tools\mingw1310_64
set CLIENT_DIR=C:\Users\Mark\Music\mkgames_\client
set BUILD_DIR=%CLIENT_DIR%\build\Desktop_Qt_6_11_2_MinGW_64_bit_Release
set INSTALLER_DIR=C:\Users\Mark\Music\mkgames_\installer
set OUTPUT_DIR=%INSTALLER_DIR%\output

echo [1/3] Building MKLauncher (Release)...
cd /d "%CLIENT_DIR%"
"%QT_DIR%\bin\qt-cmake.bat" -G "MinGW Makefiles" -DCMAKE_BUILD_TYPE=Release -B "%BUILD_DIR%" -S . 2>nul
cd /d "%BUILD_DIR%"
"%MINGW_DIR%\bin\mingw32-make.exe" -j%NUMBER_OF_PROCESSORS% 2>nul
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)
echo Build complete!
echo.

echo [2/3] Creating output directory...
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
echo.

echo [3/3] Building installer with Inno Setup...
"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" "%INSTALLER_DIR%\MKLauncher.iss"
if errorlevel 1 (
    echo Installer build failed!
    pause
    exit /b 1
)
echo.

echo ========================================
echo   Done! Installer at: %OUTPUT_DIR%\MKLauncher-Setup.exe
echo ========================================
pause
