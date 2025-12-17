#!/bin/bash
# file: install-macos.sh
# version: 1.0.0
# guid: a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d

# macOS Installation Script for Magnet Handler
# This script creates an Automator app and registers it as the default magnet link handler

set -e

echo "========================================"
echo "macOS Magnet Handler Installation"
echo "========================================"
echo ""

# Check if magnet-handler binary exists
if [ ! -f "/usr/local/bin/magnet-handler" ]; then
    echo "ERROR: magnet-handler binary not found at /usr/local/bin/magnet-handler"
    echo "Please build and install the binary first:"
    echo "  cd /Users/jdfalk/repos/github.com/jdfalk/magnet-handler"
    echo "  go build -ldflags=\"-s -w\" -o magnet-handler"
    echo "  sudo cp magnet-handler /usr/local/bin/"
    exit 1
fi

echo "✓ Found magnet-handler binary at /usr/local/bin/magnet-handler"
echo ""

# Create the Automator app structure
APP_NAME="Magnet Handler"
APP_PATH="$HOME/Applications/$APP_NAME.app"
CONTENTS_DIR="$APP_PATH/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"

echo "Creating Automator app at: $APP_PATH"
echo ""

# Create directory structure
mkdir -p "$MACOS_DIR"
mkdir -p "$CONTENTS_DIR/Resources"

# Create the shell script that will be executed
cat > "$MACOS_DIR/magnet-handler.sh" << 'SCRIPT'
#!/bin/bash
/usr/local/bin/magnet-handler "$1"
SCRIPT

chmod +x "$MACOS_DIR/magnet-handler.sh"
echo "✓ Created shell script wrapper"

# Create the main applet executable
cat > "$MACOS_DIR/applet" << 'APPLET'
#!/bin/bash
exec "$( dirname "$0" )/magnet-handler.sh" "$1"
APPLET

chmod +x "$MACOS_DIR/applet"
echo "✓ Created applet executable"

# Create the Info.plist file for the app bundle
cat > "$CONTENTS_DIR/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>applet</string>
	<key>CFBundleIconFile</key>
	<string>applet</string>
	<key>CFBundleIdentifier</key>
	<string>com.magnethandler.app</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>Magnet Handler</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleURLName</key>
			<string>Magnet Link</string>
			<key>CFBundleURLSchemes</key>
			<array>
				<string>magnet</string>
			</array>
		</dict>
	</array>
	<key>NSAppleScriptEnabled</key>
	<false/>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSHumanReadableCopyright</key>
	<string>Magnet Handler for Deluge</string>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
</dict>
</plist>
PLIST

echo "✓ Created Info.plist"

# Create PkgInfo file
echo -n "APPL????" > "$CONTENTS_DIR/PkgInfo"
echo "✓ Created PkgInfo"

echo ""
echo "✓ Automator app created successfully!"
echo "  Location: $APP_PATH"
echo ""

# Register the app as the default handler for magnet links using duti
echo "Registering magnet link handler..."
echo ""

if ! command -v duti &> /dev/null; then
    echo "WARNING: duti not found. Installing via Homebrew..."
    brew install duti
fi

# Get the bundle identifier from the app
BUNDLE_ID="com.magnethandler.app"

echo "Setting $BUNDLE_ID as default handler for magnet links..."
duti -s "$BUNDLE_ID" magnet

echo "✓ Magnet protocol handler registered!"
echo ""

# Test the app
echo "Testing the handler..."
echo ""
echo "Checking if magnet links will open with Magnet Handler app..."
DEFAULT_HANDLER=$(duti magnet 2>/dev/null || echo "unknown")
echo "  Default handler for magnet://: $DEFAULT_HANDLER"
echo ""

echo "========================================"
echo "✓ Installation Complete!"
echo "========================================"
echo ""
echo "You can now:"
echo "  1. Click magnet links in Chrome/Safari"
echo "  2. They will automatically open with Magnet Handler"
echo "  3. The handler will add them to Deluge"
echo ""
echo "Configuration:"
echo "  Edit: ~/.magnet-handler.conf"
echo ""
echo "Testing:"
echo "  /usr/local/bin/magnet-handler --help"
echo ""
