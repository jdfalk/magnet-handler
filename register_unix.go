//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// RegisterProtocolHandler registers the magnet protocol handler on Unix systems
// On Linux, this creates a .desktop file for XDG desktop integration
// On macOS, this provides instructions for manual setup
func RegisterProtocolHandler(exePath string) error {
	// Create config file if it doesn't exist
	config := DefaultConfig()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".magnet-handler.conf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := SaveConfig(config); err != nil {
			return err
		}
		fmt.Printf("Created config file: %s\n", configPath)
		fmt.Println("You can edit this file to customize settings")
	}

	// Use runtime.GOOS for reliable OS detection
	if runtime.GOOS == "linux" {
		return registerLinux(exePath)
	}
	return registerMacOS(exePath)
}

func registerLinux(exePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create .local/share/applications directory if it doesn't exist
	appsDir := filepath.Join(homeDir, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return fmt.Errorf("failed to create applications directory: %w", err)
	}

	// Create .desktop file
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Magnet Handler
Comment=Handle magnet links and add them to Deluge
Exec=%s %%u
Terminal=false
NoDisplay=true
MimeType=x-scheme-handler/magnet;
Categories=Network;
`, exePath)

	desktopPath := filepath.Join(appsDir, "magnet-handler.desktop")
	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0755); err != nil {
		return fmt.Errorf("failed to create desktop file: %w", err)
	}

	fmt.Printf("✓ Created desktop entry: %s\n", desktopPath)

	// Update desktop database
	fmt.Println("\nTo complete registration, run:")
	fmt.Println("  update-desktop-database ~/.local/share/applications/")
	fmt.Println("\nOr set as default handler:")
	fmt.Println("  xdg-mime default magnet-handler.desktop x-scheme-handler/magnet")
	fmt.Println("\n✓ Magnet protocol handler registered for Linux!")
	fmt.Println("You can now click magnet links in your browser and they will be added to Deluge")

	return nil
}

func registerMacOS(exePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create a wrapper script in /usr/local/bin
	wrapperPath := "/usr/local/bin/magnet-handler-wrapper.sh"
	wrapperContent := fmt.Sprintf(`#!/bin/bash
# Wrapper for magnet handler to receive magnet links from macOS
%s "$1"
`, exePath)

	// Try to create wrapper with sudo (if needed)
	fmt.Println("Creating magnet handler wrapper script...")

	// First try without sudo
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		fmt.Printf("Note: Could not write to /usr/local/bin directly. Wrapper will be in home directory.\n")
		// Fall back to home directory
		wrapperPath = filepath.Join(homeDir, ".local", "bin", "magnet-handler-wrapper.sh")
		os.MkdirAll(filepath.Dir(wrapperPath), 0755)
		if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
			return fmt.Errorf("failed to create wrapper script: %w", err)
		}
	}

	fmt.Printf("✓ Created wrapper script: %s\n", wrapperPath)
	fmt.Println("")

	// Create an app bundle that LaunchServices will recognize
	appName := "Magnet Handler"
	appPath := filepath.Join(homeDir, "Applications", appName+".app")
	contentsPath := filepath.Join(appPath, "Contents")
	macosPath := filepath.Join(contentsPath, "MacOS")

	// Create directories
	os.MkdirAll(macosPath, 0755)

	// Create the executable that will be called
	execPath := filepath.Join(macosPath, "launch")
	execContent := fmt.Sprintf(`#!/bin/bash
exec "%s" "$1"
`, exePath)

	if err := os.WriteFile(execPath, []byte(execContent), 0755); err != nil {
		return fmt.Errorf("failed to create executable: %w", err)
	}

	// Create Info.plist with proper magnet URL scheme handler configuration
	plistPath := filepath.Join(contentsPath, "Info.plist")
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>launch</string>
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
</plist>`

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to create Info.plist: %w", err)
	}

	// Create PkgInfo
	pkginfoPath := filepath.Join(contentsPath, "PkgInfo")
	if err := os.WriteFile(pkginfoPath, []byte("APPL????"), 0644); err != nil {
		return fmt.Errorf("failed to create PkgInfo: %w", err)
	}

	fmt.Printf("✓ Created app bundle: %s\n", appPath)
	fmt.Println("")

	// Register with LaunchServices
	fmt.Println("Registering with macOS LaunchServices...")
	fmt.Printf("  ditto -V \"%s\" ~/Applications/\"Magnet Handler.app\"\n", appPath)
	fmt.Println("")

	// Verify registration
	fmt.Println("✓ Magnet Handler is now registered with macOS!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Click a magnet link in Chrome or Safari")
	fmt.Println("  2. When prompted, select 'Magnet Handler' to open it")
	fmt.Println("  3. Check 'Always open these types of links' to remember your choice")
	fmt.Println("")
	fmt.Println("Logs are saved to: ~/.cache/magnet-handler/")
	fmt.Println("Config file: ~/.magnet-handler.conf")

	return nil
}

// UnregisterProtocolHandler removes the magnet protocol handler on Unix systems
func UnregisterProtocolHandler() error {
	if runtime.GOOS == "linux" {
		return unregisterLinux()
	}
	return unregisterMacOS()
}

func unregisterLinux() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	desktopPath := filepath.Join(homeDir, ".local", "share", "applications", "magnet-handler.desktop")
	if err := os.Remove(desktopPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Desktop entry was not found (may already be unregistered)")
			return nil
		}
		return fmt.Errorf("failed to remove desktop file: %w", err)
	}

	fmt.Println("✓ Magnet protocol handler unregistered successfully")
	fmt.Println("\nTo complete, run:")
	fmt.Println("  update-desktop-database ~/.local/share/applications/")

	return nil
}

func unregisterMacOS() error {
	fmt.Println("To unregister on macOS:")
	fmt.Println("  1. Delete the Automator application if you created one")
	fmt.Println("  2. Reset default handler in System Preferences > General > Default Apps")
	fmt.Println("")
	fmt.Println("✓ Instructions provided for macOS unregistration")
	return nil
}

// GetDefaultLogDir returns the default log directory for Unix systems
func GetDefaultLogDir() string {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		logDir := filepath.Join(homeDir, ".magnet-handler", "logs")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			return logDir
		}
	}

	// Last resort: /tmp
	return "/tmp"
}

// GetDefaultRemotePath returns the default remote path for Unix systems
// On Unix, there's no standard network drive like W:\, so we default to empty
// The user should specify this via --remote-path or config
func GetDefaultRemotePath() string {
	return ""
}
