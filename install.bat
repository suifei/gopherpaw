@echo off
setlocal EnableDelayedExpansion

:: Colors for output
set "GREEN=[92m"
set "YELLOW=[93m"
set "RED=[91m"
set "NC=[0m"

:: Print functions
goto :main

:print_info
echo %GREEN%[INFO]%NC% %~1
goto :eof

:print_warn
echo %YELLOW%[WARN]%NC% %~1
goto :eof

:print_error
echo %RED%[ERROR]%NC% %~1
goto :eof

:main

:: Check for Go
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    call :print_error "Go is not installed. Please install Go 1.23 or later."
    call :print_info "Visit: https://golang.org/doc/install"
    exit /b 1
)

:: Get Go version
for /f "tokens=3" %%v in ('go version') do set GO_VERSION=%%v
call :print_info "Go version: %GO_VERSION%"

:: Set installation directory
set "INSTALL_DIR=%USERPROFILE%\.local\bin"
set "CONFIG_DIR=%USERPROFILE%\.gopherpaw"
set "DATA_DIR=%USERPROFILE%\.gopherpaw"

:: Create directories
call :print_info "Creating directories..."
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%CONFIG_DIR%" mkdir "%CONFIG_DIR%"
if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"

:: Build from source
call :print_info "Building GopherPaw from source..."
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

go build -o gopherpaw.exe .\cmd\gopherpaw\

if %ERRORLEVEL% neq 0 (
    call :print_error "Build failed!"
    exit /b 1
)

:: Install binary
call :print_info "Installing GopherPaw to %INSTALL_DIR%..."
copy /Y gopherpaw.exe "%INSTALL_DIR%\gopherpaw.exe" >nul

:: Create default config if not exists
if not exist "%CONFIG_DIR%\config.yaml" (
    call :print_info "Creating default configuration..."
    if exist "configs\config.yaml.example" (
        copy /Y "configs\config.yaml.example" "%CONFIG_DIR%\config.yaml" >nul
    )
)

:: Create active_skills and customized_skills directories
if not exist "%CONFIG_DIR%\active_skills" mkdir "%CONFIG_DIR%\active_skills"
if not exist "%CONFIG_DIR%\customized_skills" mkdir "%CONFIG_DIR%\customized_skills"

:: Check if INSTALL_DIR is in PATH
echo %PATH% | findstr /C:"%INSTALL_DIR%" >nul
if %ERRORLEVEL% neq 0 (
    call :print_warn "%INSTALL_DIR% is not in your PATH."
    call :print_info "Add the following directory to your PATH:"
    echo.
    echo     %INSTALL_DIR%
    echo.
)

:: Print success message
call :print_info "GopherPaw installed successfully!"
call :print_info "Binary location: %INSTALL_DIR%\gopherpaw.exe"
call :print_info "Config directory: %CONFIG_DIR%"
call :print_info "Data directory: %DATA_DIR%"
echo.
call :print_info "To get started, run:"
echo     gopherpaw --help

endlocal
