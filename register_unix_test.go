//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Test GetDefaultLogDir on Unix
func TestGetDefaultLogDirUnix(t *testing.T) {
	logDir := GetDefaultLogDir()
	
	// Should not be empty
	if logDir == "" {
		t.Error("GetDefaultLogDir should not return empty string")
	}
	
	// Should be an absolute path or /tmp
	if !filepath.IsAbs(logDir) && logDir != "/tmp" {
		t.Errorf("GetDefaultLogDir should return absolute path, got %q", logDir)
	}
}

// Test GetDefaultLogDir respects XDG_CACHE_HOME
func TestGetDefaultLogDirXDG(t *testing.T) {
	// Save and restore original XDG_CACHE_HOME
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)
	
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "xdg-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Set XDG_CACHE_HOME
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	
	logDir := GetDefaultLogDir()
	
	// Should be under the XDG_CACHE_HOME
	expectedPrefix := filepath.Join(tmpDir, "magnet-handler")
	if logDir != expectedPrefix {
		t.Errorf("With XDG_CACHE_HOME=%q, expected log dir %q, got %q", tmpDir, expectedPrefix, logDir)
	}
}

// Test GetDefaultRemotePath on Unix returns empty
func TestGetDefaultRemotePathUnix(t *testing.T) {
	remotePath := GetDefaultRemotePath()
	
	// On Unix, this should return empty string
	if remotePath != "" {
		t.Errorf("GetDefaultRemotePath on Unix should return empty string, got %q", remotePath)
	}
}

// Test registerLinux creates desktop file
func TestRegisterLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}
	
	// Create temp home directory
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Override HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Call registerLinux
	err = registerLinux("/usr/bin/magnet-handler")
	if err != nil {
		t.Fatalf("registerLinux failed: %v", err)
	}
	
	// Verify desktop file was created
	desktopPath := filepath.Join(tmpDir, ".local", "share", "applications", "magnet-handler.desktop")
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		t.Fatal("Desktop file was not created")
	}
	
	// Verify content
	content, err := os.ReadFile(desktopPath)
	if err != nil {
		t.Fatalf("Failed to read desktop file: %v", err)
	}
	
	contentStr := string(content)
	
	// Check required fields
	if !strings.Contains(contentStr, "[Desktop Entry]") {
		t.Error("Desktop file missing [Desktop Entry] header")
	}
	if !strings.Contains(contentStr, "Type=Application") {
		t.Error("Desktop file missing Type=Application")
	}
	if !strings.Contains(contentStr, "MimeType=x-scheme-handler/magnet;") {
		t.Error("Desktop file missing MimeType for magnet links")
	}
	if !strings.Contains(contentStr, "Exec=/usr/bin/magnet-handler") {
		t.Error("Desktop file missing correct Exec path")
	}
}

// Test unregisterLinux removes desktop file
func TestUnregisterLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}
	
	// Create temp home directory
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Override HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create the desktop file first
	appsDir := filepath.Join(tmpDir, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatalf("Failed to create apps dir: %v", err)
	}
	
	desktopPath := filepath.Join(appsDir, "magnet-handler.desktop")
	if err := os.WriteFile(desktopPath, []byte("[Desktop Entry]\n"), 0644); err != nil {
		t.Fatalf("Failed to create desktop file: %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		t.Fatal("Desktop file was not created for test")
	}
	
	// Call unregisterLinux
	err = unregisterLinux()
	if err != nil {
		t.Fatalf("unregisterLinux failed: %v", err)
	}
	
	// Verify file was removed
	if _, err := os.Stat(desktopPath); !os.IsNotExist(err) {
		t.Error("Desktop file should have been removed")
	}
}

// Test unregisterLinux handles missing file gracefully
func TestUnregisterLinuxMissingFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}
	
	// Create temp home directory (without desktop file)
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Override HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Call unregisterLinux - should not error for missing file
	err = unregisterLinux()
	if err != nil {
		t.Errorf("unregisterLinux should handle missing file gracefully, got error: %v", err)
	}
}

// Test RegisterProtocolHandler dispatches correctly on Linux
func TestRegisterProtocolHandlerLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}
	
	// Create temp home directory
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Override HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Call RegisterProtocolHandler
	err = RegisterProtocolHandler("/usr/bin/magnet-handler")
	if err != nil {
		t.Fatalf("RegisterProtocolHandler failed: %v", err)
	}
	
	// Verify config file was created
	configPath := filepath.Join(tmpDir, ".magnet-handler.conf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
	
	// Verify desktop file was created
	desktopPath := filepath.Join(tmpDir, ".local", "share", "applications", "magnet-handler.desktop")
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		t.Error("Desktop file was not created")
	}
}

// Test registerMacOS doesn't error (just prints instructions)
func TestRegisterMacOS(t *testing.T) {
	// registerMacOS just prints instructions and returns nil
	err := registerMacOS("/usr/local/bin/magnet-handler")
	if err != nil {
		t.Errorf("registerMacOS should not error, got: %v", err)
	}
}

// Test unregisterMacOS doesn't error
func TestUnregisterMacOS(t *testing.T) {
	// unregisterMacOS just prints instructions and returns nil
	err := unregisterMacOS()
	if err != nil {
		t.Errorf("unregisterMacOS should not error, got: %v", err)
	}
}
