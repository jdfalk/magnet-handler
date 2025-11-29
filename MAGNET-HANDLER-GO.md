# Magnet Handler - Secure Self-Contained Executable

## Overview

This is a compiled Go application that safely handles magnet: protocol links from Chrome. It features:
- **Security**: Strict input validation, no code execution possible
- **Performance**: Local database with smart remote sync (no network delays)
- **Reliability**: Conflict-free replication with sequence numbers
- **Concurrent-Safe**: Multiple clicks won't corrupt data

## Security Features

1. **No Code Interpretation**: Compiled binary, not an interpreter
2. **Strict Input Validation**: Uses regex whitelist - only allows safe characters in magnet URIs
3. **Type Safety**: Go's type system prevents injection attacks
4. **No Shell Execution**: Direct HTTP API calls only
5. **Self-Contained**: Single .exe file, no dependencies

## Smart Sync System

The handler uses a local-first database with intelligent syncing:

- **Local Storage**: `~/.magnet-list.json` (fast, no network delays)
- **Remote Storage**: `W:\magnet-list.json` (backup, synced automatically)
- **Conflict Resolution**: Sequence numbers track all changes, merge intelligently
- **Best Effort Sync**: Remote sync failures don't block operations

## Building

```powershell
# Install Go from https://go.dev/dl/

# Build the executable
cd C:\Users\jdfal\repos\temp
go mod download
go build -o magnet-handler.exe magnet-handler.go

# The result is a single magnet-handler.exe file
```

## Installation

```powershell
# Run as Administrator
.\magnet-handler.exe --register
```

This will:
- Create config file at `C:\Users\<you>\.magnet-handler.conf`
- Register the protocol handler in Windows registry
- Chrome will now use this executable for magnet: links

## Configuration

Edit `C:\Users\<you>\.magnet-handler.conf`:

```json
{
  "deluge_host": "172.16.2.30",
  "deluge_port": "8112",
  "deluge_password": "deluge",
  "deluge_label": "audiobooks",
  "json_path": "W:\\magnet-list.json"
}
```

## Usage

Once registered, clicking magnet links in Chrome will automatically:
1. Validate the magnet URI (strict whitelist)
2. Connect to Deluge securely
3. Add the torrent with label
4. Log to `%TEMP%\magnet-handler-<pid>.log`

## Validation Rules

The executable only accepts magnet URIs that:
- Start with `magnet:?`
- Contain only: `a-zA-Z0-9:?&=%-.~+_`
- Include `xt=urn:btih:` parameter

Any attempt to inject shell commands, Python code, or malicious input is rejected.

## Uninstall

```powershell
# Run as Administrator
.\magnet-handler.exe --unregister
```

## Why This Is Secure

| Python Version                    | Go Version                         |
| --------------------------------- | ---------------------------------- |
| Interpreter runs arbitrary code   | Compiled binary, no interpretation |
| String concatenation for commands | Type-safe API calls only           |
| subprocess.run() shell execution  | Direct HTTP requests               |
| Limited input validation          | Strict regex whitelist validation  |
| Multiple file dependencies        | Single .exe file                   |

The Go version eliminates the entire class of injection vulnerabilities by:
1. Not using a scripting language interpreter
2. Validating ALL input with strict whitelisting
3. Making direct API calls (no shell/subprocess)
4. Using Go's type safety to prevent injection
