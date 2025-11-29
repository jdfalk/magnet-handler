# Build script for magnet-handler
# Run this to create the secure self-contained executable

Write-Host "Building secure magnet handler..." -ForegroundColor Cyan

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "✓ Found: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Go is not installed!" -ForegroundColor Red
    Write-Host "  Download from: https://go.dev/dl/" -ForegroundColor Yellow
    exit 1
}

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Cyan
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to download dependencies" -ForegroundColor Red
    exit 1
}

# Build the executable
Write-Host "`nBuilding executable..." -ForegroundColor Cyan
go build -ldflags="-s -w" -o magnet-handler.exe magnet-handler.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "`n✓ Build successful!" -ForegroundColor Green
Write-Host "`nCreated: magnet-handler.exe" -ForegroundColor Cyan

$size = (Get-Item magnet-handler.exe).Length / 1MB
Write-Host ("Size: {0:N2} MB" -f $size) -ForegroundColor Cyan

Write-Host "`nNext steps:" -ForegroundColor Yellow
Write-Host "1. Run as Administrator: .\magnet-handler.exe --register" -ForegroundColor White
Write-Host "2. Click a magnet link in Chrome to test" -ForegroundColor White
Write-Host "`nThis replaces the Python version with a secure compiled executable." -ForegroundColor Green
