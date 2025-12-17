#!/bin/bash
# file: test-magnet-handler.sh
# version: 1.0.0
# guid: b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e

# Test script to verify magnet handler is working correctly

MAGNET_LINKS=(
    "magnet:?xt=urn:btih:f8c7f48dce4dc65d83f6d68c58b47797c6afc9ea&dn=Ubuntu-22.04"
    "magnet:?xt=urn:btih:e3811b9539cacff680e418124272177c47477157&dn=Debian-12"
)

echo "Testing Magnet Handler Installation"
echo "===================================="
echo ""

# Check if binary exists
if [ ! -f /usr/local/bin/magnet-handler ]; then
    echo "ERROR: /usr/local/bin/magnet-handler not found!"
    exit 1
fi

echo "✓ Magnet handler binary found"
echo ""

# Check config
if [ ! -f ~/.magnet-handler.conf ]; then
    echo "ERROR: ~/.magnet-handler.conf not found!"
    exit 1
fi

echo "✓ Configuration file found"
CONFIG=$(cat ~/.magnet-handler.conf)
echo "  Config: $CONFIG"
echo ""

# Check log directory
mkdir -p ~/.cache/magnet-handler
echo "✓ Log directory ready at ~/.cache/magnet-handler/"
echo ""

# Test with a magnet link
echo "Testing with magnet link..."
echo "${MAGNET_LINKS[0]}"
echo ""

/usr/local/bin/magnet-handler "${MAGNET_LINKS[0]}"

echo ""
echo "Check the logs:"
ls -lht ~/.cache/magnet-handler/magnet-handler-*.log | head -1
echo ""
echo "View the latest log:"
LATEST_LOG=$(ls -t ~/.cache/magnet-handler/magnet-handler-*.log 2>/dev/null | head -1)
if [ -n "$LATEST_LOG" ]; then
    echo "  tail -20 $LATEST_LOG"
    echo ""
    tail -20 "$LATEST_LOG" | grep -E "Successfully added|Processing|Error|Authenticated"
fi
