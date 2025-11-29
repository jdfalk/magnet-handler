//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Check if we're on Linux or macOS
	if isLinux() {
		return registerLinux(exePath)
	}
	return registerMacOS(exePath)
}

func isLinux() bool {
	// Check for Linux-specific paths
	_, err := os.Stat("/usr/share/applications")
	return err == nil
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
	fmt.Println("macOS Protocol Handler Setup")
	fmt.Println("=============================")
	fmt.Println("")
	fmt.Printf("Executable path: %s\n", exePath)
	fmt.Println("")
	fmt.Println("To register the magnet handler on macOS:")
	fmt.Println("")
	fmt.Println("Option 1: Create an Automator Application")
	fmt.Println("  1. Open Automator and create a new Application")
	fmt.Println("  2. Add 'Run Shell Script' action")
	fmt.Printf("  3. Enter: %s \"$1\"\n", exePath)
	fmt.Println("  4. Save as 'Magnet Handler' in Applications")
	fmt.Println("  5. Open the app once, then set it as default for magnet links")
	fmt.Println("")
	fmt.Println("Option 2: Use a third-party tool like 'duti' or 'SwiftDefaultApps'")
	fmt.Println("  brew install duti")
	fmt.Println("  duti -s com.your.magnethandler magnet")
	fmt.Println("")
	fmt.Println("The handler is ready to use from the command line:")
	fmt.Printf("  %s \"magnet:?xt=urn:btih:...\"\n", exePath)

	return nil
}

// UnregisterProtocolHandler removes the magnet protocol handler on Unix systems
func UnregisterProtocolHandler() error {
	if isLinux() {
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
	// Try XDG_CACHE_HOME first
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir != "" {
		logDir := filepath.Join(cacheDir, "magnet-handler")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			return logDir
		}
	}

	// Fall back to ~/.cache/magnet-handler
	homeDir, err := os.UserHomeDir()
	if err == nil {
		logDir := filepath.Join(homeDir, ".cache", "magnet-handler")
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
