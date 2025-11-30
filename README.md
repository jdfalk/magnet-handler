# Magnet Handler for Deluge

Secure magnet link protocol handler for Deluge torrent client. Replaces Python-based handlers with a compiled Go executable for enhanced security.

## Features

- ✅ **Secure**: Compiled Go binary with strict URI validation
- ✅ **UUID-based tracking**: Prevents duplicate entries across systems
- ✅ **Local-first database**: Fast, reliable local storage with network sync
- ✅ **Multi-version parser**: Migrates from legacy Python formats
- ✅ **Data safety**: Critical checks prevent data loss
- ✅ **Network sync**: Automatic sync with network storage (best effort)
- ✅ **Backfill support**: Import existing Deluge torrents
- ✅ **Retry queue**: Automatically retry failed additions
- ✅ **Orphan detection**: Clean up entries no longer in Deluge

## Installation

### Quick Install (Windows)

```powershell
# Run PowerShell as Administrator
cd path\to\magnet-handler

# Install to Program Files
.\install-to-program-files.ps1

# Close and reopen PowerShell as Administrator, then register
.\reregister-from-program-files.ps1
```

### Manual Setup

```powershell
# Build from source
.\build-handler.ps1

# Register protocol handler (run as Administrator)
.\magnet-handler.exe --register

# Configure settings
.\magnet-handler.exe --host 192.168.1.100 --password YOUR_PASSWORD --save-settings
```

## Configuration

Settings are stored in `~/.magnet-handler.conf` (JSON format).

### Command-line Configuration

```powershell
# Set Deluge server details
magnet-handler.exe --host 192.168.1.100 --port 8112 --password deluge --save-settings

# Set torrent label
magnet-handler.exe --label audiobooks --save-settings

# Set network storage path
magnet-handler.exe --remote-path "W:\magnet-list-network.json" --save-settings
```

### Configuration File

Default: `~/.magnet-handler.conf`

```json
{
  "deluge_host": "192.168.1.100",
  "deluge_port": "8112",
  "deluge_password": "deluge",
  "deluge_label": "audiobooks",
  "json_path": "C:\\Users\\YourName\\magnet-list-local.json",
  "remote_path": "W:\\magnet-list-network.json"
}
```

## Usage

### Protocol Handler

Once registered, clicking magnet links in your browser automatically:
1. Validates the magnet URI
2. Adds to Deluge with your configured label
3. Saves to local database
4. Syncs with network storage

### Command-line Operations

```powershell
# Backfill from existing Deluge torrents
magnet-handler.exe --backfill

# Process retry queue
magnet-handler.exe --retry

# Sync database with Deluge (dry run)
magnet-handler.exe --sync-dry-run

# Sync database with Deluge (remove orphans)
magnet-handler.exe --sync

# Migrate database formats
magnet-handler.exe --migrate

# Show version
magnet-handler.exe --version

# Unregister protocol handler
magnet-handler.exe --unregister
```

## Database Files

- **Local**: `~/magnet-list-local.json` - Fast, always available
- **Network**: Configurable (e.g., `W:\magnet-list-network.json`) - Backup sync

The handler automatically:
- Saves to local first (reliable)
- Syncs with network storage (best effort)
- Compares checksums to detect conflicts
- Merges changes intelligently

## Development

### Building

```powershell
# Clean build
.\build-handler.ps1
```

### Testing

```powershell
# Test with a magnet link
.\magnet-handler.exe "magnet:?xt=urn:btih:HASH&dn=Name"
```

## CI/CD

This project uses reusable workflows from [jdfalk/ghcommon](https://github.com/jdfalk/ghcommon):

- **CI**: Automatic builds and tests on push/PR
- **Release**: Multi-platform binary builds on version tags

Create a release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

## Security

- **No Python interpreter**: Prevents remote code execution
- **Strict validation**: Regex-based URI validation
- **Compiled binary**: No script injection possible
- **Hash verification**: SHA1 checksums for sync detection

## Migration from Python

The handler automatically migrates from the old Python format:

```powershell
# Copy old file and migrate
Copy-Item "old-magnet-list.json" "~/magnet-list-local.json"
magnet-handler.exe --migrate
```

All fields preserved:
- Title, Hash, URI
- Timestamps (first_seen, last_attempt, added_date)
- Status, TorrentID, AddedToDeluge
- SavePath, TorrentName, RetryCount

## License

MIT License - see LICENSE file for details

## Author

Built for secure, reliable magnet link handling with Deluge.
