//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// RegisterProtocolHandler registers the magnet protocol handler in Windows registry
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

	// Register protocol handler
	k, _, err := registry.CreateKey(registry.CLASSES_ROOT, `magnet`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetStringValue("", "URL:Magnet Protocol"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	// Set default icon
	k2, _, err := registry.CreateKey(registry.CLASSES_ROOT, `magnet\DefaultIcon`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k2.Close()
	if err := k2.SetStringValue("", fmt.Sprintf("%s,0", exePath)); err != nil {
		return err
	}

	// Set command
	k3, _, err := registry.CreateKey(registry.CLASSES_ROOT, `magnet\shell\open\command`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k3.Close()

	command := fmt.Sprintf(`"%s" "%%1"`, exePath)
	if err := k3.SetStringValue("", command); err != nil {
		return err
	}

	fmt.Println("✓ Magnet protocol handler registered successfully!")
	fmt.Println("You can now click magnet links in Chrome and they will be added to Deluge")
	return nil
}

// UnregisterProtocolHandler removes the magnet protocol handler
func UnregisterProtocolHandler() error {
	if err := registry.DeleteKey(registry.CLASSES_ROOT, `magnet\shell\open\command`); err != nil {
		return err
	}
	if err := registry.DeleteKey(registry.CLASSES_ROOT, `magnet\shell\open`); err != nil {
		return err
	}
	if err := registry.DeleteKey(registry.CLASSES_ROOT, `magnet\shell`); err != nil {
		return err
	}
	if err := registry.DeleteKey(registry.CLASSES_ROOT, `magnet\DefaultIcon`); err != nil {
		return err
	}
	if err := registry.DeleteKey(registry.CLASSES_ROOT, `magnet`); err != nil {
		return err
	}

	fmt.Println("✓ Magnet protocol handler unregistered successfully")
	return nil
}

// GetDefaultLogDir returns the default log directory for Windows
func GetDefaultLogDir() string {
	logDir := os.Getenv("TEMP")
	if logDir == "" {
		logDir = "."
	}
	return logDir
}

// GetDefaultRemotePath returns the default remote path for Windows
func GetDefaultRemotePath() string {
	return "W:\\magnet-list-network.json"
}
