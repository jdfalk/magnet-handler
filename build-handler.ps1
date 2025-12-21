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
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "INSTALLATION REQUIRED" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "The installer will now launch and request administrator privileges." -ForegroundColor White
Write-Host "Please accept the UAC prompt to install to Program Files." -ForegroundColor White
Write-Host "`nPress any key to continue..." -ForegroundColor Cyan
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

$sourcePath = Join-Path $PWD "magnet-handler.exe"
$installScript = Join-Path $PSScriptRoot "install.ps1"
$destPath = "C:\Program Files\MagnetHandler\magnet-handler.exe"

try {
    $process = Start-Process powershell.exe -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File","`"$installScript`"","-SourcePath","`"$sourcePath`"" -Verb RunAs -PassThru -Wait

    # Check if the installation succeeded
    if (Test-Path $destPath) {
        $destVersion = (Get-Item $destPath).LastWriteTime
        $sourceVersion = (Get-Item $sourcePath).LastWriteTime

        if ($destVersion -ge $sourceVersion) {
            Write-Host "`n✓ Installation successful!" -ForegroundColor Green
            Write-Host "Installed to: $destPath" -ForegroundColor Cyan
        } else {
            Write-Host "`n⚠ Warning: Installed file appears older than source" -ForegroundColor Yellow
            Write-Host "You may need to manually copy the file" -ForegroundColor Yellow
        }
    } else {
        Write-Host "`n⚠ Warning: Could not verify installation" -ForegroundColor Yellow
        Write-Host "If UAC was canceled or failed, manually run as Administrator:" -ForegroundColor Yellow
        Write-Host "Copy-Item `"$sourcePath`" `"$destPath`" -Force" -ForegroundColor White
    }
} catch {
    Write-Host "`n✗ Installation failed or was canceled" -ForegroundColor Red
    Write-Host "To install manually, run PowerShell as Administrator and execute:" -ForegroundColor Yellow
    Write-Host "Copy-Item `"$sourcePath`" `"$destPath`" -Force" -ForegroundColor White
}

Write-Host "`nNext steps:" -ForegroundColor Yellow
Write-Host "1. Close and reopen PowerShell, then run: magnet-handler.exe --register" -ForegroundColor White
Write-Host "2. Click a magnet link in Chrome to test" -ForegroundColor White
Write-Host "`nThis replaces the Python version with a secure compiled executable." -ForegroundColor Green
