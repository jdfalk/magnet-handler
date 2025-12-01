# Build script for magnet-handler
# Run this to create the secure self-contained executable

Write-Host "Building secure magnet handler..." -ForegroundColor Cyan

# Check if Go is installed
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Go is not installed!" -ForegroundColor Red
    Write-Host "Download from: https://go.dev/dl/" -ForegroundColor Yellow
    pause
    exit 1
}
Write-Host "Found: $goVersion" -ForegroundColor Green

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Cyan
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to download dependencies" -ForegroundColor Red
    pause
    exit 1
}

# Clean previous build
Write-Host "`nCleaning previous build..." -ForegroundColor Cyan
go clean

# Build the executable
Write-Host "`nBuilding executable..." -ForegroundColor Cyan
go build -ldflags="-s -w" -o magnet-handler.exe
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Build failed" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "`nBuild successful!" -ForegroundColor Green
Write-Host "Created: magnet-handler.exe" -ForegroundColor Cyan

$size = (Get-Item magnet-handler.exe).Length / 1MB
Write-Host ("Size: {0:N2} MB" -f $size) -ForegroundColor Cyan

# Run install script with elevation
Write-Host "`nLaunching installer (will request admin privileges)..." -ForegroundColor Cyan
$sourcePath = Join-Path $PWD "magnet-handler.exe"
$installScript = Join-Path $PSScriptRoot "install.ps1"
Start-Process powershell.exe -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File","`"$installScript`"","-SourcePath","`"$sourcePath`"" -Verb RunAs -Wait

Write-Host "`nNext steps:" -ForegroundColor Yellow
Write-Host "1. Close and reopen PowerShell, then run: magnet-handler.exe --register" -ForegroundColor White
Write-Host "2. Click a magnet link in Chrome to test" -ForegroundColor White
Write-Host "`nThis replaces the Python version with a secure compiled executable." -ForegroundColor Green
