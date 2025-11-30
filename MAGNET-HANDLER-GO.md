# Magnet Handler - Cross-Platform Secure Executable

## Overview

This is a compiled Go application that safely handles magnet: protocol links from your browser. It features:
- **Cross-Platform**: Works on Windows, Linux, and macOS
- **Security**: Strict input validation, no code execution possible
- **Performance**: Local database with smart remote sync (no network delays)
- **Reliability**: Conflict-free replication with sequence numbers
- **Concurrent-Safe**: Multiple clicks won't corrupt data

## Security Features

1. **No Code Interpretation**: Compiled binary, not an interpreter
2. **Strict Input Validation**: Uses regex whitelist - only allows safe characters in magnet URIs
3. **Type Safety**: Go's type system prevents injection attacks
4. **No Shell Execution**: Direct HTTP API calls only
5. **Self-Contained**: Single executable file, no dependencies

## Smart Sync System

The handler uses a local-first database with intelligent syncing:

- **Local Storage**: `~/magnet-list-local.json` (fast, no network delays)
- **Remote Storage**: Configurable path (e.g., NAS, network share, cloud mount)
- **Conflict Resolution**: Sequence numbers track all changes, merge intelligently
- **Best Effort Sync**: Remote sync failures don't block operations

## Building

### All Platforms

```bash
# Install Go from https://go.dev/dl/

# Clone and build
git clone <repo-url>
cd magnet-handler
go mod download
go build -o magnet-handler   # Linux/macOS
go build -o magnet-handler.exe  # Windows
```

### Cross-Compilation

```bash
# Build for all platforms from any OS
GOOS=linux GOARCH=amd64 go build -o magnet-handler-linux
GOOS=darwin GOARCH=amd64 go build -o magnet-handler-mac
GOOS=windows GOARCH=amd64 go build -o magnet-handler.exe
```

## Installation

### Linux

```bash
# Register as protocol handler
./magnet-handler --register

# Complete registration (one of these):
update-desktop-database ~/.local/share/applications/
# or
xdg-mime default magnet-handler.desktop x-scheme-handler/magnet
```

### macOS

```bash
# Get instructions for macOS setup
./magnet-handler --register
```

### Windows (Run as Administrator)

```powershell
.\magnet-handler.exe --register
```

## Configuration

The config file is located at `~/.magnet-handler.conf`:

```json
{
  "deluge_host": "192.168.0.1",
  "deluge_port": "8112",
  "deluge_password": "deluge",
  "deluge_label": "audiobooks",
  "json_path": "/home/user/magnet-list-local.json",
  "remote_path": "/mnt/nas/magnet-list-network.json"
}
```

### Setting Remote Path from Command Line

```bash
# Use a remote path for this session only
./magnet-handler --remote-path /mnt/nas/magnet-list.json "magnet:?xt=urn:btih:..."

# Save the remote path to config for future use
./magnet-handler --remote-path /mnt/nas/magnet-list.json --save-path

# Windows example with network drive
magnet-handler.exe --remote-path "W:\magnet-list.json" --save-path
```

## Usage

### Basic Usage

Once registered, clicking magnet links in your browser will automatically:
1. Validate the magnet URI (strict whitelist)
2. Connect to Deluge securely
3. Add the torrent with label
4. Log activity

### Command Line Options

```
-register        Register as magnet protocol handler
-unregister      Unregister magnet protocol handler
-remote-path     Path to shared/network storage for syncing
-save-path       Save the remote-path to config file
-backfill        Backfill database from existing Deluge torrents
-retry           Process all items in retry queue
-migrate         Migrate JSON files to new format
-version         Show version
```

### Log Locations

- **Linux**: `~/.cache/magnet-handler/magnet-handler-<pid>.log`
- **macOS**: `~/.cache/magnet-handler/magnet-handler-<pid>.log`
- **Windows**: `%TEMP%\magnet-handler-<pid>.log`

## Validation Rules

The executable only accepts magnet URIs that:
- Start with `magnet:?`
- Contain only: `a-zA-Z0-9:?&=%-.~+_`
- Include `xt=urn:btih:` parameter

Any attempt to inject shell commands or malicious input is rejected.

## Uninstall

### Linux

```bash
./magnet-handler --unregister
update-desktop-database ~/.local/share/applications/
```

### macOS

```bash
./magnet-handler --unregister
# Follow the displayed instructions
```

### Windows (Run as Administrator)

```powershell
.\magnet-handler.exe --unregister
```

## Why This Is Secure

| Python Version                    | Go Version                         |
| --------------------------------- | ---------------------------------- |
| Interpreter runs arbitrary code   | Compiled binary, no interpretation |
| String concatenation for commands | Type-safe API calls only           |
| subprocess.run() shell execution  | Direct HTTP requests               |
| Limited input validation          | Strict regex whitelist validation  |
| Multiple file dependencies        | Single executable file             |

The Go version eliminates the entire class of injection vulnerabilities by:
1. Not using a scripting language interpreter
2. Validating ALL input with strict whitelisting
3. Making direct API calls (no shell/subprocess)
4. Using Go's type safety to prevent injection
