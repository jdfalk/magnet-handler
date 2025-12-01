# Build script for magnet-handler
# Run this to create the secure self-contained executable

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "Requesting administrator privileges..." -ForegroundColor Yellow
    $scriptCmd = "cd '$PWD'; & '$PSCommandPath'; Write-Host '`nPress Enter to exit...' -ForegroundColor Gray; Read-Host"
    Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -Command `"$scriptCmd`"" -Verb RunAs
    exit
}

Write-Host "Building secure magnet handler..." -ForegroundColor Cyan

# Check if Go is installed
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Go is not installed!" -ForegroundColor Red
    Write-Host "  Download from: https://go.dev/dl/" -ForegroundColor Yellow
    pause
    exit 1
}
Write-Host "✓ Found: $goVersion" -ForegroundColor Green

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Cyan
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to download dependencies" -ForegroundColor Red
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
    Write-Host "✗ Build failed" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "`n✓ Build successful!" -ForegroundColor Green
Write-Host "`nCreated: magnet-handler.exe" -ForegroundColor Cyan

$size = (Get-Item magnet-handler.exe).Length / 1MB
Write-Host ("Size: {0:N2} MB" -f $size) -ForegroundColor Cyan

# Install to Program Files
Write-Host "`nInstalling to Program Files..." -ForegroundColor Cyan
$destFolder = "C:\Program Files\MagnetHandler"
$destPath = Join-Path $destFolder "magnet-handler.exe"

if (-not (Test-Path $destFolder)) {
    New-Item -ItemType Directory -Path $destFolder -Force | Out-Null
}

Copy-Item "magnet-handler.exe" $destPath -Force

if (Test-Path $destPath) {
    Write-Host "Installed to: $destPath" -ForegroundColor Green
} else {
    Write-Host "Installation failed" -ForegroundColor Red
    pause
    exit 1
}

# Add to PATH if not already there
$currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($currentPath -notlike "*$destFolder*") {
    Write-Host "Adding to system PATH..." -ForegroundColor Cyan
    $newPath = "$currentPath;$destFolder"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
    Write-Host "Added to PATH (restart shell to use)" -ForegroundColor Green
} else {
    Write-Host "Already in system PATH" -ForegroundColor Green
}

Write-Host "`nNext steps:" -ForegroundColor Yellow
Write-Host "1. Close and reopen PowerShell, then run: magnet-handler.exe --register" -ForegroundColor White
Write-Host "2. Click a magnet link in Chrome to test" -ForegroundColor White
Write-Host "`nThis replaces the Python version with a secure compiled executable." -ForegroundColor Green
