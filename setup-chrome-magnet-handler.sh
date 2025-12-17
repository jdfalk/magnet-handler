#!/bin/bash
# file: setup-chrome-magnet-handler.sh
# version: 1.0.0
# guid: c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f

# Setup script to properly register magnet handler for Chrome on macOS
# This creates a wrapper that opens Terminal so we can see debug output

set -e

echo "=========================================="
echo "macOS Magnet Handler - Chrome Setup"
echo "=========================================="
echo ""

# Create a wrapper script that opens Terminal
WRAPPER_PATH="/usr/local/bin/magnet-handler-chrome"

cat > "$WRAPPER_PATH" << 'WRAPPER_EOF'
#!/bin/bash
# Magnet Handler wrapper for Chrome
# Opens a Terminal window to show what's happening

MAGNET_URI="$1"

if [ -z "$MAGNET_URI" ]; then
    exit 1
fi

# Create a temp script that will run in Terminal
TEMP_SCRIPT=$(mktemp /tmp/magnet-handler-XXXX.sh)

cat > "$TEMP_SCRIPT" << 'SCRIPT'
#!/bin/bash
MAGNET_URI="$1"

# Set up colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo ""
echo "=================================================="
echo "  Magnet Handler Invoked from Chrome"
echo "=================================================="
echo ""
echo -e "${YELLOW}Magnet URI:${NC}"
echo "$MAGNET_URI"
echo ""
echo -e "${YELLOW}Processing...${NC}"
echo ""

# Call the handler
LOG_DIR="$HOME/.cache/magnet-handler"
mkdir -p "$LOG_DIR"

/usr/local/bin/magnet-handler "$MAGNET_URI"
EXIT_CODE=$?

echo ""
echo "=================================================="
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ Success!${NC}"
else
    echo -e "${RED}✗ Failed with exit code: $EXIT_CODE${NC}"
fi
echo "=================================================="
echo ""
echo "Logs saved to: $LOG_DIR/"
echo ""
echo "Press Enter to close this window..."
read

SCRIPT

chmod +x "$TEMP_SCRIPT"

# Open Terminal with the script, passing the magnet URI
open -a Terminal "$TEMP_SCRIPT" -- "$MAGNET_URI"

# Clean up temp script after a delay
(sleep 5; rm -f "$TEMP_SCRIPT") &
WRAPPER_EOF

chmod +x "$WRAPPER_PATH"

echo "✓ Created wrapper at: $WRAPPER_PATH"
echo ""

# Now update the app bundle to use this wrapper
APP_PATH="$HOME/Applications/Magnet Handler.app"
LAUNCH_SCRIPT="$APP_PATH/Contents/MacOS/launch"

cat > "$LAUNCH_SCRIPT" << 'LAUNCH_EOF'
#!/bin/bash
# Launch script for Magnet Handler app bundle
# Receives magnet URI and passes it to the wrapper

/usr/local/bin/magnet-handler-chrome "$1"
exit 0
LAUNCH_EOF

chmod +x "$LAUNCH_SCRIPT"

echo "✓ Updated app bundle launch script"
echo ""

# Update Info.plist to ensure it's properly configured
PLIST="$APP_PATH/Contents/Info.plist"

# Verify URL scheme is registered
if grep -q "magnet" "$PLIST"; then
    echo "✓ URL scheme 'magnet' is registered in Info.plist"
else
    echo "⚠ Warning: magnet URL scheme not found in Info.plist"
fi

echo ""
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "Now when you click a magnet link in Chrome:"
echo ""
echo "1. Chrome will ask 'How do you want to open this?'"
echo "2. Look for 'Magnet Handler' in the list"
echo "3. Select it and check 'Always use this app'"
echo "4. A Terminal window will open showing what's happening"
echo "5. The magnet will be added to Deluge"
echo ""
echo "App location: $APP_PATH"
echo "Wrapper: $WRAPPER_PATH"
echo "Logs: ~/.cache/magnet-handler/"
echo ""
echo "To test manually:"
echo "  $WRAPPER_PATH \"magnet:?xt=urn:btih:f8c7f48dce4dc65d83f6d68c58b47797c6afc9ea&dn=Ubuntu\""
echo ""
