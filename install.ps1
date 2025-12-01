# Install script - runs elevated to copy to Program Files
# Called automatically by build-handler.ps1

param([string]$SourcePath)

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "ERROR: Install script must run as Administrator" -ForegroundColor Red
    pause
    exit 1
}

if (-not $SourcePath -or -not (Test-Path $SourcePath)) {
    Write-Host "ERROR: Source file not found: $SourcePath" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "Installing to Program Files..." -ForegroundColor Cyan
$destFolder = "C:\Program Files\MagnetHandler"
$destPath = Join-Path $destFolder "magnet-handler.exe"

if (-not (Test-Path $destFolder)) {
    New-Item -ItemType Directory -Path $destFolder -Force | Out-Null
}

Copy-Item $SourcePath $destPath -Force

if (Test-Path $destPath) {
    Write-Host "Installed to: $destPath" -ForegroundColor Green
} else {
    Write-Host "ERROR: Installation failed" -ForegroundColor Red
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

Write-Host "`nInstallation complete!" -ForegroundColor Green
Write-Host "Press Enter to exit..." -ForegroundColor Gray
Read-Host
