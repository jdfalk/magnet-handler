# Magnet Link Handler Setup Guide

This guide shows how to set up automatic handling of magnet links in Chrome.

## Overview

- **Main Script**: `add-magnets-to-deluge.py` - Adds magnets to Deluge with all features
- **Handler Script**: `magnet-handler.py` - Registers as Windows protocol handler for magnet links
- **Centralized JSON**: `W:\magnet-list.json` - Single source of truth for all magnet tracking

## Setup Steps

### 1. Copy the JSON File

Copy your existing `magent-test-list.json` to the centralized location:

```powershell
Copy-Item magent-test-list.json W:\magnet-list.json
```

### 2. Register the Magnet Protocol Handler

**IMPORTANT: Run PowerShell as Administrator**

```powershell
python magnet-handler.py --register
```

You should see:
```
✓ Magnet protocol handler registered successfully!
You can now click magnet links in Chrome and they will be added to Deluge
```

### 3. Configure Chrome (if needed)

1. Click a magnet link in Chrome
2. Chrome will ask "Open magnet-handler.py?"
3. Check "Always open these types of links"
4. Click "Open link"

### 4. Test It

Click any magnet link in Chrome - it should:
1. Automatically capture the magnet link
2. Add it to Deluge via the script
3. Log everything to `W:\magnet-handler.log`

## Configuration

Edit `magnet-handler.py` to change settings:

```python
# Configuration
DELUGE_HOST = "172.16.2.30"
DELUGE_PORT = "8112"
DELUGE_PASSWORD = "deluge"
DELUGE_LABEL = "audiobooks"
```

## Usage Examples

### Using the Main Script Directly

From anywhere on your system (uses centralized JSON):

```bash
python add-magnets-to-deluge.py magnets.txt --host 172.16.2.30
```

### Backfill Existing Torrents

```bash
python add-magnets-to-deluge.py magnets.txt --host 172.16.2.30 --backfill
```

### Remove Completed Torrents

```bash
python add-magnets-to-deluge.py magnets.txt --host 172.16.2.30 --remove-completed
```

### Unpause Incomplete Torrents

```bash
python add-magnets-to-deluge.py magnets.txt --host 172.16.2.30 --unpause
```

### Click Magnet Links in Chrome

Just click - it's automatic! Check the log:

```bash
cat W:\magnet-handler.log
```

## File Locations

- **Centralized JSON**: `W:\magnet-list.json` (or `\\172.16.2.30\bigdata\books\magnet-list.json`)
- **Handler Log**: `W:\magnet-handler.log`
- **Temp Magnets**: `W:\temp-magnet.txt` (auto-cleaned)

## Troubleshooting

### Handler Not Working

1. Check if registered:
   ```powershell
   reg query HKEY_CLASSES_ROOT\magnet\shell\open\command
   ```

2. Check the log:
   ```powershell
   Get-Content W:\magnet-handler.log -Tail 50
   ```

3. Re-register:
   ```powershell
   python magnet-handler.py --unregister
   python magnet-handler.py --register
   ```

### JSON File Access Issues

Make sure the W: drive is mapped:

```powershell
net use W: \\172.16.2.30\bigdata
```

Or the script will try to create the directory automatically.

### Permission Errors

The `--register` command requires Administrator privileges:

1. Right-click PowerShell
2. Choose "Run as Administrator"
3. Run the register command

## Uninstall

To remove the protocol handler:

```powershell
python magnet-handler.py --unregister
```

## How It Works

1. **Chrome clicks magnet link** → Calls `magnet-handler.py`
2. **Handler saves magnet** → Creates temp file with magnet URI
3. **Calls main script** → Runs `add-magnets-to-deluge.py`
4. **Script processes** → Checks JSON, deduplicates, adds to Deluge
5. **Updates JSON** → Saves to centralized `W:\magnet-list.json`
6. **Cleanup** → Deletes temp file

All logging goes to `W:\magnet-handler.log` for debugging.
