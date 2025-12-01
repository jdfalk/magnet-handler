package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test ValidateMagnetURI
func TestValidateMagnetURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{
			name:     "valid magnet URI with 40-char hash",
			uri:      "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&dn=Test+File",
			expected: true,
		},
		{
			name:     "valid magnet URI with 32-char hash",
			uri:      "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&dn=Test",
			expected: true,
		},
		{
			name:     "valid magnet URI with encoded characters",
			uri:      "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&dn=Test%20File",
			expected: true,
		},
		{
			name:     "invalid - missing magnet prefix",
			uri:      "http://example.com",
			expected: false,
		},
		{
			name:     "invalid - missing xt parameter",
			uri:      "magnet:?dn=Test+File",
			expected: false,
		},
		{
			name:     "invalid - empty string",
			uri:      "",
			expected: false,
		},
		{
			name:     "invalid - contains shell injection",
			uri:      "magnet:?xt=urn:btih:aaa;rm -rf /",
			expected: false,
		},
		{
			name:     "invalid - contains backticks",
			uri:      "magnet:?xt=urn:btih:aaa`id`",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMagnetURI(tt.uri)
			if result != tt.expected {
				t.Errorf("ValidateMagnetURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

// Test ExtractMagnetHash
func TestExtractMagnetHash(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "extract 40-char hash",
			uri:      "magnet:?xt=urn:btih:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&dn=Test",
			expected: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			name:     "extract 32-char hash",
			uri:      "magnet:?xt=urn:btih:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&dn=Test",
			expected: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			name:     "mixed case hash",
			uri:      "magnet:?xt=urn:btih:AaBbCcDdEeFf1234567890AaBbCcDdEeFf123456&dn=Test",
			expected: "aabbccddeeff1234567890aabbccddeeff123456",
		},
		{
			name:     "no hash found",
			uri:      "magnet:?dn=Test",
			expected: "",
		},
		{
			name:     "empty string",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMagnetHash(tt.uri)
			if result != tt.expected {
				t.Errorf("ExtractMagnetHash(%q) = %q, expected %q", tt.uri, result, tt.expected)
			}
		})
	}
}

// Test ExtractMagnetName
func TestExtractMagnetName(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "simple name with plus for space",
			uri:      "magnet:?xt=urn:btih:aaa&dn=Test+File+Name",
			expected: "Test File Name",
		},
		{
			name:     "URL encoded name",
			uri:      "magnet:?xt=urn:btih:aaa&dn=Test%20File%20Name",
			expected: "Test File Name",
		},
		{
			name:     "name with special chars",
			uri:      "magnet:?xt=urn:btih:aaa&dn=Test%27s%20File",
			expected: "Test's File",
		},
		{
			name:     "no name parameter",
			uri:      "magnet:?xt=urn:btih:aaa",
			expected: "Unknown",
		},
		{
			name:     "empty string",
			uri:      "",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMagnetName(tt.uri)
			if result != tt.expected {
				t.Errorf("ExtractMagnetName(%q) = %q, expected %q", tt.uri, result, tt.expected)
			}
		})
	}
}

// Test DefaultConfig
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DelugeHost == "" {
		t.Error("DefaultConfig().DelugeHost should not be empty")
	}
	if config.DelugePort == "" {
		t.Error("DefaultConfig().DelugePort should not be empty")
	}
	if config.DelugePassword == "" {
		t.Error("DefaultConfig().DelugePassword should not be empty")
	}
	if config.JSONPath == "" {
		t.Error("DefaultConfig().JSONPath should not be empty")
	}
	// RemotePath can be empty on Unix systems
}

// Test GetRemotePath
func TestGetRemotePath(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectEmpty bool
	}{
		{
			name:        "nil config returns default",
			config:      nil,
			expectEmpty: GetDefaultRemotePath() == "",
		},
		{
			name: "config with remote path",
			config: &Config{
				RemotePath: "/mnt/nas/magnet-list.json",
			},
			expectEmpty: false,
		},
		{
			name: "config with empty remote path",
			config: &Config{
				RemotePath: "",
			},
			expectEmpty: GetDefaultRemotePath() == "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRemotePath(tt.config)
			if tt.expectEmpty && result != "" {
				t.Errorf("GetRemotePath() = %q, expected empty string", result)
			}
			if !tt.expectEmpty && result == "" {
				t.Error("GetRemotePath() returned empty string, expected non-empty")
			}
			// If config has a specific RemotePath, it should be returned
			if tt.config != nil && tt.config.RemotePath != "" {
				if result != tt.config.RemotePath {
					t.Errorf("GetRemotePath() = %q, expected %q", result, tt.config.RemotePath)
				}
			}
		})
	}
}

// Test SaveConfig and LoadConfig
func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary home directory for testing
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create test config
	testConfig := Config{
		DelugeHost:     "192.168.1.100",
		DelugePort:     "9999",
		DelugePassword: "testpass",
		DelugeLabel:    "testlabel",
		JSONPath:       "/test/path.json",
		RemotePath:     "/remote/test.json",
	}

	// Save config
	if err := SaveConfig(testConfig); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".magnet-handler.conf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Load config
	loadedConfig, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config matches saved config
	if loadedConfig.DelugeHost != testConfig.DelugeHost {
		t.Errorf("DelugeHost: got %q, want %q", loadedConfig.DelugeHost, testConfig.DelugeHost)
	}
	if loadedConfig.DelugePort != testConfig.DelugePort {
		t.Errorf("DelugePort: got %q, want %q", loadedConfig.DelugePort, testConfig.DelugePort)
	}
	if loadedConfig.DelugePassword != testConfig.DelugePassword {
		t.Errorf("DelugePassword: got %q, want %q", loadedConfig.DelugePassword, testConfig.DelugePassword)
	}
	if loadedConfig.DelugeLabel != testConfig.DelugeLabel {
		t.Errorf("DelugeLabel: got %q, want %q", loadedConfig.DelugeLabel, testConfig.DelugeLabel)
	}
	if loadedConfig.JSONPath != testConfig.JSONPath {
		t.Errorf("JSONPath: got %q, want %q", loadedConfig.JSONPath, testConfig.JSONPath)
	}
	if loadedConfig.RemotePath != testConfig.RemotePath {
		t.Errorf("RemotePath: got %q, want %q", loadedConfig.RemotePath, testConfig.RemotePath)
	}
}

// Test LoadConfig with missing file returns default
func TestLoadConfigMissingFile(t *testing.T) {
	// Create a temporary home directory for testing
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for testing (both Unix HOME and Windows USERPROFILE)
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	// Load config - should return default since file doesn't exist
	// Note: LoadConfig returns default config (not error) when file is missing
	config, err := LoadConfig()
	if err != nil {
		t.Logf("LoadConfig returned error (expected for missing file): %v", err)
	}

	// Should match default config structure
	defaultConfig := DefaultConfig()
	if config.DelugeHost != defaultConfig.DelugeHost {
		t.Errorf("Expected default DelugeHost %q, got %q", defaultConfig.DelugeHost, config.DelugeHost)
	}
}

// Test ComputeChecksum
func TestComputeChecksum(t *testing.T) {
	db1 := &MagnetDatabase{
		Added: map[string]MagnetEntry{
			"hash1": {Hash: "hash1", Title: "Test1"},
		},
		Retry: map[string]MagnetEntry{},
	}

	db2 := &MagnetDatabase{
		Added: map[string]MagnetEntry{
			"hash1": {Hash: "hash1", Title: "Test1"},
		},
		Retry: map[string]MagnetEntry{},
	}

	db3 := &MagnetDatabase{
		Added: map[string]MagnetEntry{
			"hash2": {Hash: "hash2", Title: "Test2"},
		},
		Retry: map[string]MagnetEntry{},
	}

	checksum1 := ComputeChecksum(db1)
	checksum2 := ComputeChecksum(db2)
	checksum3 := ComputeChecksum(db3)

	if checksum1 == "" {
		t.Error("ComputeChecksum returned empty string")
	}

	if checksum1 != checksum2 {
		t.Error("Identical databases should have same checksum")
	}

	if checksum1 == checksum3 {
		t.Error("Different databases should have different checksums")
	}
}

// Test LoadJSONDatabase with empty file
func TestLoadJSONDatabaseEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with non-existent file
	dbPath := filepath.Join(tmpDir, "nonexistent.json")
	db, err := LoadJSONDatabase(dbPath)
	if err != nil {
		t.Fatalf("LoadJSONDatabase should not error for non-existent file: %v", err)
	}

	if db == nil {
		t.Fatal("LoadJSONDatabase returned nil for non-existent file")
	}

	if len(db.Added) != 0 {
		t.Errorf("Expected empty Added map, got %d entries", len(db.Added))
	}

	if len(db.Retry) != 0 {
		t.Errorf("Expected empty Retry map, got %d entries", len(db.Retry))
	}
}

// Test LoadJSONDatabase with current format
func TestLoadJSONDatabaseCurrentFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test database in current format
	testDB := MagnetDatabase{
		Metadata: DatabaseMetadata{
			LastSequence: 2,
			LastModified: "2024-01-01T00:00:00Z",
			Checksum:     "test",
		},
		Added: map[string]MagnetEntry{
			"hash1": {ID: 1, Hash: "hash1", Title: "Test1"},
		},
		Retry: map[string]MagnetEntry{
			"hash2": {ID: 2, Hash: "hash2", Title: "Test2"},
		},
	}

	dbPath := filepath.Join(tmpDir, "test.json")
	data, _ := json.MarshalIndent(testDB, "", "  ")
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load and verify
	db, err := LoadJSONDatabase(dbPath)
	if err != nil {
		t.Fatalf("LoadJSONDatabase failed: %v", err)
	}

	if len(db.Added) != 1 {
		t.Errorf("Expected 1 Added entry, got %d", len(db.Added))
	}

	if len(db.Retry) != 1 {
		t.Errorf("Expected 1 Retry entry, got %d", len(db.Retry))
	}

	if db.Added["hash1"].Title != "Test1" {
		t.Errorf("Expected Title 'Test1', got %q", db.Added["hash1"].Title)
	}
}

// Test SaveDatabaseLocal
func TestSaveDatabaseLocal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db := &MagnetDatabase{
		Metadata: DatabaseMetadata{},
		Added: map[string]MagnetEntry{
			"hash1": {ID: 1, Hash: "hash1", Title: "Test1"},
		},
		Retry: map[string]MagnetEntry{},
	}

	dbPath := filepath.Join(tmpDir, "test.json")
	if err := SaveDatabaseLocal(dbPath, db); err != nil {
		t.Fatalf("SaveDatabaseLocal failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file was not created")
	}

	// Load and verify
	loadedDB, err := LoadJSONDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to load saved database: %v", err)
	}

	if len(loadedDB.Added) != 1 {
		t.Errorf("Expected 1 Added entry, got %d", len(loadedDB.Added))
	}

	if loadedDB.Added["hash1"].Title != "Test1" {
		t.Errorf("Expected Title 'Test1', got %q", loadedDB.Added["hash1"].Title)
	}

	// Verify metadata was updated
	if loadedDB.Metadata.LastModified == "" {
		t.Error("LastModified should be set after save")
	}

	if loadedDB.Metadata.Checksum == "" {
		t.Error("Checksum should be set after save")
	}
}

// Test MergeDatabases
func TestMergeDatabases(t *testing.T) {
	local := &MagnetDatabase{
		Metadata: DatabaseMetadata{LastSequence: 2},
		Added: map[string]MagnetEntry{
			"hash1": {ID: 1, Hash: "hash1", Title: "Local1"},
			"hash2": {ID: 2, Hash: "hash2", Title: "Local2"},
		},
		Retry: map[string]MagnetEntry{},
	}

	remote := &MagnetDatabase{
		Metadata: DatabaseMetadata{LastSequence: 3},
		Added: map[string]MagnetEntry{
			"hash1": {ID: 1, Hash: "hash1", Title: "Remote1"},
			"hash3": {ID: 3, Hash: "hash3", Title: "Remote3"},
		},
		Retry: map[string]MagnetEntry{},
	}

	merged := MergeDatabases(local, remote)

	// Should have all unique hashes
	expectedHashes := map[string]bool{"hash1": true, "hash2": true, "hash3": true}
	for hash := range expectedHashes {
		if _, exists := merged.Added[hash]; !exists {
			t.Errorf("Expected hash %q in merged database", hash)
		}
	}

	if len(merged.Added) != 3 {
		t.Errorf("Expected 3 entries in merged database, got %d", len(merged.Added))
	}

	// Metadata should be updated
	if merged.Metadata.LastModified == "" {
		t.Error("Merged database should have LastModified set")
	}

	if merged.Metadata.Checksum == "" {
		t.Error("Merged database should have Checksum set")
	}
}

// Test NewDelugeClient
func TestNewDelugeClient(t *testing.T) {
	client := NewDelugeClient("192.168.1.100", "8112", "password")

	if client.Host != "192.168.1.100" {
		t.Errorf("Expected Host '192.168.1.100', got %q", client.Host)
	}

	if client.Port != "8112" {
		t.Errorf("Expected Port '8112', got %q", client.Port)
	}

	if client.Password != "password" {
		t.Errorf("Expected Password 'password', got %q", client.Password)
	}

	expectedURL := "http://192.168.1.100:8112/json"
	if client.BaseURL != expectedURL {
		t.Errorf("Expected BaseURL %q, got %q", expectedURL, client.BaseURL)
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

// Test GetDefaultLogDir
func TestGetDefaultLogDir(t *testing.T) {
	logDir := GetDefaultLogDir()
	if logDir == "" {
		t.Error("GetDefaultLogDir() should not return empty string")
	}
}

// Test GetDefaultRemotePath
func TestGetDefaultRemotePath(t *testing.T) {
	// This is platform-specific, just ensure it doesn't panic
	_ = GetDefaultRemotePath()
	// On Unix, this returns empty string
	// On Windows, this returns "W:\magnet-list-network.json"
}

// Test Config JSON serialization with RemotePath
func TestConfigJSONSerialization(t *testing.T) {
	config := Config{
		DelugeHost:     "localhost",
		DelugePort:     "8112",
		DelugePassword: "deluge",
		DelugeLabel:    "test",
		JSONPath:       "/path/to/local.json",
		RemotePath:     "/path/to/remote.json",
	}

	// Serialize
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Deserialize
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify RemotePath is preserved
	if loaded.RemotePath != config.RemotePath {
		t.Errorf("RemotePath not preserved: got %q, want %q", loaded.RemotePath, config.RemotePath)
	}
}

// Test Config JSON serialization with empty RemotePath (omitempty)
func TestConfigJSONSerializationEmptyRemotePath(t *testing.T) {
	config := Config{
		DelugeHost:     "localhost",
		DelugePort:     "8112",
		DelugePassword: "deluge",
		DelugeLabel:    "test",
		JSONPath:       "/path/to/local.json",
		RemotePath:     "", // Empty
	}

	// Serialize
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Verify remote_path behavior with omitempty
	// Note: omitempty will exclude the field entirely if empty
	jsonStr := string(data)
	if strings.Contains(jsonStr, `"remote_path":""`) {
		t.Error("Empty remote_path should be omitted from JSON due to omitempty tag")
	}

	// Deserialize
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Empty remote path should deserialize as empty
	if loaded.RemotePath != "" {
		t.Errorf("Empty RemotePath should deserialize as empty, got %q", loaded.RemotePath)
	}
}

// Test ComputeFileChecksum
func TestComputeFileChecksum(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magnet-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testPath := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	if err := os.WriteFile(testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compute checksum
	checksum1, err := ComputeFileChecksum(testPath)
	if err != nil {
		t.Fatalf("ComputeFileChecksum failed: %v", err)
	}

	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Compute again - should be same
	checksum2, err := ComputeFileChecksum(testPath)
	if err != nil {
		t.Fatalf("ComputeFileChecksum failed: %v", err)
	}

	if checksum1 != checksum2 {
		t.Error("Same file should have same checksum")
	}

	// Modify file - should be different
	if err := os.WriteFile(testPath, []byte("different content"), 0644); err != nil {
		t.Fatalf("Failed to write modified test file: %v", err)
	}

	checksum3, err := ComputeFileChecksum(testPath)
	if err != nil {
		t.Fatalf("ComputeFileChecksum failed: %v", err)
	}

	if checksum1 == checksum3 {
		t.Error("Different file content should have different checksum")
	}

	// Non-existent file should error
	_, err = ComputeFileChecksum(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("ComputeFileChecksum should error for non-existent file")
	}
}
