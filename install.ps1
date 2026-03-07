#Requires -Version 5.1

param(
    [string]$InstallDir = "$env:USERPROFILE\.local\bin",
    [string]$ConfigDir = "$env:USERPROFILE\.gopherpaw",
    [string]$DataDir = "$env:USERPROFILE\.gopherpaw"
)

$ErrorActionPreference = "Stop"

# Print functions
function Print-Info {
    param([string]$Message)
    Write-Host "[INFO] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Print-Warn {
    param([string]$Message)
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Print-Error {
    param([string]$Message)
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

# Check for Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Print-Error "Go is not installed. Please install Go 1.23 or later."
    Print-Info "Visit: https://golang.org/doc/install"
    exit 1
}

$GoVersion = (go version) -replace 'go version go', '' -replace ' .*', ''
Print-Info "Go version: $GoVersion"

# Create directories
Print-Info "Creating directories..."
$DirsToCreate = @($InstallDir, $ConfigDir, $DataDir)
foreach ($Dir in $DirsToCreate) {
    if (-not (Test-Path $Dir)) {
        New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    }
}

# Build from source
Print-Info "Building GopherPaw from source..."
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

& go build -o gopherpaw.exe .\cmd\gopherpaw\

if ($LASTEXITCODE -ne 0) {
    Print-Error "Build failed!"
    exit 1
}

# Install binary
Print-Info "Installing GopherPaw to $InstallDir..."
Copy-Item -Path "gopherpaw.exe" -Destination "$InstallDir\gopherpaw.exe" -Force

# Create default config if not exists
$ConfigFile = Join-Path $ConfigDir "config.yaml"
if (-not (Test-Path $ConfigFile)) {
    Print-Info "Creating default configuration..."
    $ExampleConfig = Join-Path $ScriptDir "configs\config.yaml.example"
    if (Test-Path $ExampleConfig) {
        Copy-Item -Path $ExampleConfig -Destination $ConfigFile -Force
    }
}

# Create active_skills and customized_skills directories
$ActiveSkillsDir = Join-Path $ConfigDir "active_skills"
$CustomizedSkillsDir = Join-Path $ConfigDir "customized_skills"
if (-not (Test-Path $ActiveSkillsDir)) {
    New-Item -ItemType Directory -Path $ActiveSkillsDir -Force | Out-Null
}
if (-not (Test-Path $CustomizedSkillsDir)) {
    New-Item -ItemType Directory -Path $CustomizedSkillsDir -Force | Out-Null
}

# Check if InstallDir is in PATH
$PathDirs = $env:PATH -split ';'
if ($PathDirs -notcontains $InstallDir) {
    Print-Warn "$InstallDir is not in your PATH."
    Print-Info "Add the following directory to your PATH:"
    Write-Host ""
    Write-Host "    $InstallDir" -ForegroundColor Cyan
    Write-Host ""
    Print-Info "Or run the following command to add it permanently:"
    Write-Host ""
    Write-Host '    [Environment]::SetEnvironmentVariable("PATH", "$([Environment]::GetEnvironmentVariable(\"PATH\", \"User\"));'"$InstallDir'", "User")' -ForegroundColor Cyan
    Write-Host ""
}

# Print success message
Print-Info "GopherPaw installed successfully!"
Print-Info "Binary location: $InstallDir\gopherpaw.exe"
Print-Info "Config directory: $ConfigDir"
Print-Info "Data directory: $DataDir"
Write-Host ""
Print-Info "To get started, run:"
Write-Host "    gopherpaw --help" -ForegroundColor Cyan
