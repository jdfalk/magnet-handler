package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const version = "1.0.1"

// Config represents the handler configuration
type Config struct {
	DelugeHost     string `json:"deluge_host"`
	DelugePort     string `json:"deluge_port"`
	DelugePassword string `json:"deluge_password"`
	DelugeLabel    string `json:"deluge_label"`
	JSONPath       string `json:"json_path"`
	RemotePath     string `json:"remote_path,omitempty"` // Path to shared/network storage (optional)
}

// MagnetEntry represents a tracked magnet link
type MagnetEntry struct {
	UUID          string `json:"uuid"`         // Unique UUID (preferred)
	ID            int64  `json:"id,omitempty"` // Deprecated: old sequence ID for migration
	Title         string `json:"title"`
	Hash          string `json:"hash"`
	URI           string `json:"uri"`
	AddedDate     string `json:"added_date"`
	FirstSeen     string `json:"first_seen,omitempty"`      // When first encountered
	LastAttempt   string `json:"last_attempt,omitempty"`    // Last time we tried to add
	Status        string `json:"status,omitempty"`          // success/failed
	TorrentID     string `json:"torrent_id,omitempty"`      // Deluge's torrent ID
	AddedToDeluge string `json:"added_to_deluge,omitempty"` // When Deluge accepted it
	RetryCount    int    `json:"retry_count,omitempty"`
	SavePath      string `json:"save_path,omitempty"`
	TorrentName   string `json:"torrent_name,omitempty"`
}

// DatabaseMetadata tracks sync state
type DatabaseMetadata struct {
	LastSequence int64  `json:"last_sequence"` // Highest ID assigned
	LastModified string `json:"last_modified"` // Timestamp of last write
	Checksum     string `json:"checksum"`      // Hash of added+retry for conflict detection
}

// MagnetDatabase represents the JSON structure (current version)
type MagnetDatabase struct {
	Metadata DatabaseMetadata       `json:"metadata"`
	Added    map[string]MagnetEntry `json:"added"` // Successfully added or duplicates
	Retry    map[string]MagnetEntry `json:"retry"` // Failed, needs retry
}

// Legacy formats for migration

// V0: Python version - flat map of hash->entry (no added/retry wrapper)
type MagnetEntryV0 struct {
	Hash          string  `json:"hash"`
	Title         string  `json:"title"`
	URI           string  `json:"uri"`
	TorrentName   string  `json:"torrent_name"`
	SavePath      string  `json:"save_path"`
	State         string  `json:"state,omitempty"`
	Progress      float64 `json:"progress,omitempty"`
	Status        string  `json:"status,omitempty"`
	TorrentID     string  `json:"torrent_id,omitempty"`
	AddedToDeluge string  `json:"added_to_deluge,omitempty"`
	FirstSeen     string  `json:"first_seen,omitempty"`
	LastAttempt   string  `json:"last_attempt,omitempty"`
	Backfilled    string  `json:"backfilled,omitempty"`
}

// V1: First Go version with added/retry but no IDs
type MagnetEntryV1 struct {
	Title       string `json:"title"`
	Hash        string `json:"hash"`
	URI         string `json:"uri"`
	AddedDate   string `json:"added_date"`
	LastAttempt string `json:"last_attempt,omitempty"`
	RetryCount  int    `json:"retry_count,omitempty"`
	SavePath    string `json:"save_path,omitempty"`
	TorrentName string `json:"torrent_name,omitempty"`
}

type MagnetDatabaseV1 struct {
	Added map[string]MagnetEntryV1 `json:"added"`
	Retry map[string]MagnetEntryV1 `json:"retry"`
}

type DatabaseMetadataV2 struct {
	LastSequence int64  `json:"last_sequence"`
	LastModified string `json:"last_modified"`
	Checksum     string `json:"checksum"`
}

type MagnetDatabaseV2 struct {
	Metadata DatabaseMetadataV2     `json:"metadata"`
	Added    map[string]MagnetEntry `json:"added"`
	Retry    map[string]MagnetEntry `json:"retry"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	homeDir, _ := getHomeDir()
	return Config{
		DelugeHost:     "192.168.0.1",
		DelugePort:     "8112",
		DelugePassword: "deluge",
		DelugeLabel:    "audiobooks",
		JSONPath:       filepath.Join(homeDir, "magnet-list-local.json"), // Local by default
		RemotePath:     GetDefaultRemotePath(),                           // Platform-specific default
	}
}

// GetRemotePath returns the remote path for the database from config
// This is now configurable instead of hardcoded to W:\
func GetRemotePath(config *Config) string {
	if config != nil && config.RemotePath != "" {
		return config.RemotePath
	}
	return GetDefaultRemotePath()
}

// ComputeChecksum generates SHA1 hash of database contents
func ComputeChecksum(db *MagnetDatabase) string {
	// Marshal to JSON for consistent hashing
	data, err := json.Marshal(db)
	if err != nil {
		return ""
	}

	// Compute SHA1 hash
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// ComputeFileChecksum computes SHA1 hash of file contents on disk
func ComputeFileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

// GenerateUUID generates a RFC4122 v4 UUID
func GenerateUUID() string {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	// Set version 4 (random)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant RFC4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// MigrateFileFormat migrates a JSON file to the new format with proper checksums
func MigrateFileFormat(path string) error {
	log.Printf("Migrating file format: %s", path)

	// Compute original file checksum BEFORE loading
	oldChecksum := ""
	if _, err := os.Stat(path); err == nil {
		oldChecksum, _ = ComputeFileChecksum(path)
		log.Printf("Original file checksum: %s", oldChecksum)
	}

	// Load the file (will handle legacy format)
	db, err := LoadJSONDatabase(path)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	log.Printf("Loaded: %d added, %d retry, last_sequence=%d",
		len(db.Added), len(db.Retry), db.Metadata.LastSequence)

	// Update checksum and metadata
	db.Metadata.Checksum = ComputeChecksum(db)
	// Generate UUIDs for entries that don't have them
	uuidsGenerated := 0
	for hash, entry := range db.Added {
		if entry.UUID == "" {
			entry.UUID = GenerateUUID()
			db.Added[hash] = entry
			uuidsGenerated++
		}
	}
	for hash, entry := range db.Retry {
		if entry.UUID == "" {
			entry.UUID = GenerateUUID()
			db.Retry[hash] = entry
			uuidsGenerated++
		}
	}
	if uuidsGenerated > 0 {
		log.Printf("Generated %d UUIDs for existing entries", uuidsGenerated)
	}

	db.Metadata.LastModified = time.Now().Format(time.RFC3339)

	// Save with new format
	if err := SaveDatabaseLocal(path, db); err != nil {
		return fmt.Errorf("failed to save migrated file: %w", err)
	}

	// Compute new file checksum AFTER saving
	newFileChecksum, _ := ComputeFileChecksum(path)

	log.Printf("✓ Migrated successfully")
	log.Printf("  Data checksum: %s", db.Metadata.Checksum)
	log.Printf("  File checksum: %s", newFileChecksum)
	log.Printf("  Last sequence: %d", db.Metadata.LastSequence)

	return nil
}

// getHomeDir returns the user's home directory, checking env vars first for testability
func getHomeDir() (string, error) {
	// Check environment variables first (for testing)
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return userProfile, nil
	}
	// Fall back to os.UserHomeDir()
	return os.UserHomeDir()
}

// LoadConfig loads configuration from file
func LoadConfig() (Config, error) {
	homeDir, err := getHomeDir()
	if err != nil {
		return DefaultConfig(), err
	}

	configPath := filepath.Join(homeDir, ".magnet-handler.conf")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), err
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config Config) error {
	homeDir, err := getHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".magnet-handler.conf")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// ValidateMagnetURI strictly validates a magnet URI to prevent injection
func ValidateMagnetURI(uri string) bool {
	// Must start with magnet:?
	if !strings.HasPrefix(uri, "magnet:?") {
		return false
	}

	// Must contain xt parameter with btih hash
	// Only allow alphanumeric, :, ?, &, =, %, -, _, ., ~, +
	// This is a strict whitelist to prevent any injection
	validPattern := regexp.MustCompile(`^magnet:\?[a-zA-Z0-9:?&=%\-_.~+]+$`)
	if !validPattern.MatchString(uri) {
		return false
	}

	// Must have xt parameter
	if !strings.Contains(uri, "xt=urn:btih:") {
		return false
	}

	return true
}

// ExtractMagnetHash extracts the info hash from a magnet URI
func ExtractMagnetHash(uri string) string {
	// Find xt=urn:btih: parameter
	re := regexp.MustCompile(`xt=urn:btih:([a-fA-F0-9]{40}|[a-zA-Z0-9]{32})`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return ""
}

// ExtractMagnetName extracts the display name from a magnet URI
func ExtractMagnetName(uri string) string {
	// Find dn= parameter
	re := regexp.MustCompile(`dn=([^&]+)`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) > 1 {
		// URL decode the name - handle all common encodings
		name := matches[1]
		// First replace + with space
		name = strings.ReplaceAll(name, "+", " ")
		// Then decode hex sequences
		decoded := ""
		i := 0
		for i < len(name) {
			if name[i] == '%' && i+2 < len(name) {
				// Try to decode hex
				if hexVal := name[i+1 : i+3]; len(hexVal) == 2 {
					var b byte
					if _, err := fmt.Sscanf(hexVal, "%02x", &b); err == nil {
						decoded += string(b)
						i += 3
						continue
					}
				}
			}
			decoded += string(name[i])
			i++
		}
		return decoded
	}
	return "Unknown"
}

// LoadJSONDatabase loads the JSON database file with retry logic
func LoadJSONDatabase(path string) (*MagnetDatabase, error) {
	db := &MagnetDatabase{
		Metadata: DatabaseMetadata{},
		Added:    make(map[string]MagnetEntry),
		Retry:    make(map[string]MagnetEntry),
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return db, nil
	}

	// Try multiple times with backoff
	for attempt := 0; attempt < 5; attempt++ {
		data, err := os.ReadFile(path)
		if err != nil {
			if attempt < 4 {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			return db, err
		}

		// Try to parse as current format first
		err = json.Unmarshal(data, db)
		if err == nil && (len(db.Added) > 0 || len(db.Retry) > 0) {
			// Successfully parsed and got entries
			return db, nil
		}

		// Try V0 format (Python version - flat map of hash->entry)
		dbV0 := make(map[string]MagnetEntryV0)
		errV0 := json.Unmarshal(data, &dbV0)
		if errV0 == nil && len(dbV0) > 0 {
			nextID := int64(1)
			for hash, v0 := range dbV0 {
				// Use URI from V0 if present, otherwise construct from hash
				uri := v0.URI
				if uri == "" {
					uri = fmt.Sprintf("magnet:?xt=urn:btih:%s", v0.Hash)
				}

				// All V0 entries go into "added" since Python version didn't track retry
				db.Added[hash] = MagnetEntry{
					UUID:          GenerateUUID(),
					ID:            nextID,
					Title:         v0.Title,
					Hash:          v0.Hash,
					URI:           uri,
					AddedDate:     v0.FirstSeen,
					FirstSeen:     v0.FirstSeen,
					LastAttempt:   v0.LastAttempt,
					Status:        v0.Status,
					TorrentID:     v0.TorrentID,
					AddedToDeluge: v0.AddedToDeluge,
					SavePath:      v0.SavePath,
					TorrentName:   v0.TorrentName,
				}
				nextID++
			}
			db.Metadata.LastSequence = nextID - 1
			db.Metadata.Checksum = ComputeChecksum(db)

			log.Printf("Loaded V0 format (Python): %d entries (use --migrate)", len(db.Added))
			return db, nil
		}

		// Try V1 format (no metadata, no IDs, but has added/retry)
		dbV1 := &MagnetDatabaseV1{
			Added: make(map[string]MagnetEntryV1),
			Retry: make(map[string]MagnetEntryV1),
		}
		errV1 := json.Unmarshal(data, dbV1)
		if errV1 == nil && (len(dbV1.Added) > 0 || len(dbV1.Retry) > 0) {
			nextID := int64(1)
			for hash, v1 := range dbV1.Added {
				db.Added[hash] = MagnetEntry{
					UUID: GenerateUUID(), ID: nextID, Title: v1.Title, Hash: v1.Hash, URI: v1.URI,
					AddedDate: v1.AddedDate, LastAttempt: v1.LastAttempt,
					RetryCount: v1.RetryCount, SavePath: v1.SavePath, TorrentName: v1.TorrentName,
				}
				nextID++
			}
			for hash, v1 := range dbV1.Retry {
				db.Retry[hash] = MagnetEntry{
					UUID: GenerateUUID(), ID: nextID, Title: v1.Title, Hash: v1.Hash, URI: v1.URI,
					AddedDate: v1.AddedDate, LastAttempt: v1.LastAttempt,
					RetryCount: v1.RetryCount, SavePath: v1.SavePath, TorrentName: v1.TorrentName,
				}
				nextID++
			}
			db.Metadata.LastSequence = nextID - 1
			db.Metadata.Checksum = ComputeChecksum(db)

			totalEntries := len(db.Added) + len(db.Retry)
			if len(data) > 1024 && totalEntries == 0 {
				log.Printf("CRITICAL: V1 parsed 0 entries from %d bytes!", len(data))
				return nil, fmt.Errorf("V1 parsing failed: 0 entries")
			}

			log.Printf("Loaded V1 format: %d entries (use --migrate)", totalEntries)
			return db, nil
		}

		// SAFETY CHECK: If file is large but all parsers got 0 entries, something is very wrong
		if len(data) > 1024 {
			log.Printf("CRITICAL: File is %d bytes but all parsers got 0 entries!", len(data))
			log.Printf("Preview: %s", string(data[:min(200, len(data))]))
		}

		log.Printf("ERROR: All parsers failed")
		log.Printf("  Current format: %v", err)
		log.Printf("  V0 (Python flat map): %v, entries=%d", errV0, len(dbV0))
		log.Printf("  V1 (added/retry): %v, entries=%d", errV1, len(dbV1.Added)+len(dbV1.Retry))
		log.Printf("File size: %d bytes", len(data))
		return nil, fmt.Errorf("unrecognized format")
	}

	return db, fmt.Errorf("failed to load JSON after retries")
}

// MergeDatabases intelligently merges two databases based on sequence numbers
func MergeDatabases(local, remote *MagnetDatabase) *MagnetDatabase {
	merged := &MagnetDatabase{
		Added: make(map[string]MagnetEntry),
		Retry: make(map[string]MagnetEntry),
	}

	// Both databases will be fully merged based on IDs and timestamps

	// Merge strategy: newer IDs win, for same ID take most recent timestamp
	allHashes := make(map[string]bool)
	for hash := range local.Added {
		allHashes[hash] = true
	}
	for hash := range local.Retry {
		allHashes[hash] = true
	}
	for hash := range remote.Added {
		allHashes[hash] = true
	}
	for hash := range remote.Retry {
		allHashes[hash] = true
	}

	nextID := int64(1)
	for hash := range allHashes {
		// Check all four locations
		localAdded, inLocalAdded := local.Added[hash]
		localRetry, inLocalRetry := local.Retry[hash]
		remoteAdded, inRemoteAdded := remote.Added[hash]
		remoteRetry, inRemoteRetry := remote.Retry[hash]

		var winner MagnetEntry
		var inAdded bool

		// Priority: Added > Retry, Higher ID > Lower ID, Newer timestamp > Older
		candidates := []struct {
			entry   MagnetEntry
			isAdded bool
			exists  bool
		}{
			{localAdded, true, inLocalAdded},
			{localRetry, false, inLocalRetry},
			{remoteAdded, true, inRemoteAdded},
			{remoteRetry, false, inRemoteRetry},
		}

		winnerFound := false
		for _, c := range candidates {
			if !c.exists {
				continue
			}
			if !winnerFound || c.isAdded && !inAdded || c.entry.ID > winner.ID {
				winner = c.entry
				inAdded = c.isAdded
				winnerFound = true
			}
		}

		if winnerFound {
			// Assign new sequential ID if needed
			if winner.ID == 0 {
				winner.ID = nextID
				nextID++
			} else {
				if winner.ID >= nextID {
					nextID = winner.ID + 1
				}
			}

			if inAdded {
				merged.Added[hash] = winner
			} else {
				merged.Retry[hash] = winner
			}
		}
	}

	// Update metadata
	merged.Metadata.LastSequence = nextID - 1
	merged.Metadata.LastModified = time.Now().Format(time.RFC3339)
	merged.Metadata.Checksum = ComputeChecksum(merged)

	return merged
}

// SyncWithRemote syncs local database with remote, returns merged result
func SyncWithRemote(localPath, remotePath string) (*MagnetDatabase, error) {
	// Compute file checksums BEFORE loading/parsing
	localFileChecksum, localFileErr := ComputeFileChecksum(localPath)
	remoteFileChecksum, remoteFileErr := ComputeFileChecksum(remotePath)

	// If file checksums match, no need to merge
	if localFileErr == nil && remoteFileErr == nil && localFileChecksum == remoteFileChecksum && len(localFileChecksum) >= 8 {
		log.Printf("Files are identical (checksum: %s...), using local", localFileChecksum[:8])
		local, err := LoadJSONDatabase(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load local: %w", err)
		}
		return local, nil
	}

	// Load local
	local, err := LoadJSONDatabase(localPath)
	if err != nil {
		log.Printf("Warning: Failed to load local DB: %v", err)
		local = &MagnetDatabase{
			Added: make(map[string]MagnetEntry),
			Retry: make(map[string]MagnetEntry),
		}
	}

	// Try to load remote
	remote, err := LoadJSONDatabase(remotePath)
	if err != nil {
		log.Printf("Remote DB not accessible, using local only")
		return local, nil
	}

	// Files differ, need to merge
	localPreview := "empty"
	remotePreview := "empty"
	if len(localFileChecksum) >= 8 {
		localPreview = localFileChecksum[:8] + "..."
	}
	if len(remoteFileChecksum) >= 8 {
		remotePreview = remoteFileChecksum[:8] + "..."
	}
	log.Printf("Files differ: local=%s remote=%s", localPreview, remotePreview)

	log.Printf("Merging: Local(seq=%d) + Remote(seq=%d)", local.Metadata.LastSequence, remote.Metadata.LastSequence)
	merged := MergeDatabases(local, remote)
	log.Printf("Merged: %d added, %d retry (seq=%d)", len(merged.Added), len(merged.Retry), merged.Metadata.LastSequence)

	return merged, nil
}

// SaveDatabaseLocal saves database to local path only (fast)
func SaveDatabaseLocal(path string, db *MagnetDatabase) error {
	// Update metadata
	db.Metadata.LastModified = time.Now().Format(time.RFC3339)
	db.Metadata.Checksum = ComputeChecksum(db)

	// Write to temp file first, then rename (atomic)
	tempPath := path + ".tmp"
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(tempPath, data, 0644)
	if err != nil {
		return err
	}

	// Atomic rename
	err = os.Rename(tempPath, path)
	if err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

// SaveJSONDatabase saves database locally with smart sync logic
func SaveJSONDatabase(localPath string, updates *MagnetDatabase, config *Config) error {
	remotePath := GetRemotePath(config)

	// Load and sync with remote first
	merged, err := SyncWithRemote(localPath, remotePath)
	if err != nil {
		log.Printf("Warning: Sync failed: %v", err)
		// Try to at least load local
		merged, err = LoadJSONDatabase(localPath)
		if err != nil {
			log.Printf("Warning: Could not load local either, starting fresh")
			merged = &MagnetDatabase{
				Metadata: DatabaseMetadata{},
				Added:    make(map[string]MagnetEntry),
				Retry:    make(map[string]MagnetEntry),
			}
		}
	}

	// Safety check: if merged database is empty but remote has data, use remote
	if len(merged.Added) == 0 && len(merged.Retry) == 0 && remotePath != "" {
		log.Printf("Warning: Loaded database is empty, checking remote...")
		remote, err := LoadJSONDatabase(remotePath)
		if err == nil && (len(remote.Added) > 0 || len(remote.Retry) > 0) {
			log.Printf("Found %d entries in remote, using that instead", len(remote.Added)+len(remote.Retry))
			merged = remote
		}
	}

	// Apply updates to merged database
	nextID := merged.Metadata.LastSequence + 1
	for hash, entry := range updates.Added {
		if entry.ID == 0 {
			entry.ID = nextID
			nextID++
		}
		merged.Added[hash] = entry
		// Remove from retry if exists
		delete(merged.Retry, hash)
	}
	for hash, entry := range updates.Retry {
		if entry.ID == 0 {
			entry.ID = nextID
			nextID++
		}
		merged.Retry[hash] = entry
	}
	merged.Metadata.LastSequence = nextID - 1

	// Save locally (fast, no network)
	if err := SaveDatabaseLocal(localPath, merged); err != nil {
		return fmt.Errorf("failed to save local: %w", err)
	}
	log.Printf("Saved to local: %s", localPath)

	// Try to copy to remote (best effort, don't fail if network issue)
	if remotePath != "" {
		if err := SaveDatabaseLocal(remotePath, merged); err != nil {
			log.Printf("Warning: Could not sync to remote: %v", err)
			log.Printf("Changes saved locally, will sync on next operation")
		} else {
			log.Printf("Synced to remote: %s", remotePath)
		}
	}

	return nil
}

// DelugeClient handles communication with Deluge Web API
type DelugeClient struct {
	Host       string
	Port       string
	Password   string
	BaseURL    string
	HTTPClient *http.Client
	Cookie     string
}

// NewDelugeClient creates a new Deluge client
func NewDelugeClient(host, port, password string) *DelugeClient {
	return &DelugeClient{
		Host:     host,
		Port:     port,
		Password: password,
		BaseURL:  fmt.Sprintf("http://%s:%s/json", host, port),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest makes a JSON-RPC request to Deluge
func (c *DelugeClient) makeRequest(method string, params []interface{}) (map[string]interface{}, error) {
	requestBody := map[string]interface{}{
		"method": method,
		"params": params,
		"id":     1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Cookie != "" {
		req.Header.Set("Cookie", c.Cookie)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Save cookie from response
	if cookies := resp.Cookies(); len(cookies) > 0 {
		c.Cookie = cookies[0].String()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Authenticate logs into Deluge
func (c *DelugeClient) Authenticate() error {
	result, err := c.makeRequest("auth.login", []interface{}{c.Password})
	if err != nil {
		return err
	}

	if success, ok := result["result"].(bool); !ok || !success {
		return fmt.Errorf("authentication failed")
	}

	return nil
}

// Connect connects to Deluge daemon
func (c *DelugeClient) Connect() error {
	// Check if already connected
	result, err := c.makeRequest("web.connected", []interface{}{})
	if err != nil {
		return err
	}

	if connected, ok := result["result"].(bool); ok && connected {
		return nil
	}

	// Get hosts
	result, err = c.makeRequest("web.get_hosts", []interface{}{})
	if err != nil {
		return err
	}

	hosts, ok := result["result"].([]interface{})
	if !ok || len(hosts) == 0 {
		return fmt.Errorf("no Deluge hosts available")
	}

	// Connect to first host
	host := hosts[0].([]interface{})
	hostID := host[0].(string)

	_, err = c.makeRequest("web.connect", []interface{}{hostID})
	return err
}

// AddMagnet adds a magnet URI to Deluge
func (c *DelugeClient) AddMagnet(magnetURI, label string) error {
	// Add magnet
	result, err := c.makeRequest("core.add_torrent_magnet", []interface{}{magnetURI, map[string]interface{}{}})
	if err != nil {
		return err
	}

	// Check for error in result
	if errInfo, ok := result["error"]; ok && errInfo != nil {
		return fmt.Errorf("Deluge error: %v", errInfo)
	}

	hash, ok := result["result"].(string)
	if !ok {
		return fmt.Errorf("failed to get torrent hash from response")
	}

	// Set label if provided
	if label != "" {
		// Ensure label exists
		_, _ = c.makeRequest("label.add", []interface{}{label})
		// Ignore error if label already exists

		// Set label on torrent
		_, err = c.makeRequest("label.set_torrent", []interface{}{hash, label})
		if err != nil {
			log.Printf("Warning: Failed to set label: %v", err)
		}
	}

	return nil
}

// GetTorrentsByLabel retrieves all torrents with a specific label
func (c *DelugeClient) GetTorrentsByLabel(label string) (map[string]map[string]interface{}, error) {
	// Get all torrents with their info
	keys := []string{"name", "hash", "save_path", "label"}
	result, err := c.makeRequest("core.get_torrents_status", []interface{}{map[string]interface{}{}, keys})
	if err != nil {
		return nil, err
	}

	torrents, ok := result["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Filter by label
	filtered := make(map[string]map[string]interface{})
	for hash, torrentData := range torrents {
		torrentMap, ok := torrentData.(map[string]interface{})
		if !ok {
			continue
		}
		torrentLabel, _ := torrentMap["label"].(string)
		if torrentLabel == label {
			filtered[hash] = torrentMap
		}
	}

	return filtered, nil
}

// AddMagnetToDeluge is the main handler function
func AddMagnetToDeluge(magnetURI string, config Config) error {
	// Strict validation - no injection possible
	if !ValidateMagnetURI(magnetURI) {
		return fmt.Errorf("invalid magnet URI format")
	}

	log.Printf("Processing magnet link: %.100s...", magnetURI)

	// Extract hash and name
	hash := ExtractMagnetHash(magnetURI)
	name := ExtractMagnetName(magnetURI)

	if hash == "" {
		return fmt.Errorf("could not extract hash from magnet URI")
	}

	// Load database
	db, err := LoadJSONDatabase(config.JSONPath)
	if err != nil {
		log.Printf("Warning: Could not load database: %v", err)
		db = &MagnetDatabase{
			Added: make(map[string]MagnetEntry),
			Retry: make(map[string]MagnetEntry),
		}
	}

	// Check if already successfully added
	if _, exists := db.Added[hash]; exists {
		log.Printf("✓ Already added: %s", name)
		log.Printf("Retry queue: %d items", len(db.Retry))
		return nil
	}

	// Check if already in retry queue (don't retry automatically)
	if entry, exists := db.Retry[hash]; exists {
		log.Printf("⚠ Already in retry queue: %s", name)
		log.Printf("  Last attempt: %s (attempt #%d)", entry.LastAttempt, entry.RetryCount)
		log.Printf("  Use --retry flag to process retry queue")
		log.Printf("Retry queue: %d items", len(db.Retry))
		return nil
	}

	// Create Deluge client
	client := NewDelugeClient(config.DelugeHost, config.DelugePort, config.DelugePassword)

	// Create entry for JSON (do this first so we can save it even if connection fails)
	entry := MagnetEntry{
		UUID:        GenerateUUID(),
		Title:       name,
		Hash:        hash,
		URI:         magnetURI,
		AddedDate:   time.Now().Format(time.RFC3339),
		LastAttempt: time.Now().Format(time.RFC3339),
		RetryCount:  1,
	}

	// Prepare database update
	dbUpdate := &MagnetDatabase{
		Added: make(map[string]MagnetEntry),
		Retry: make(map[string]MagnetEntry),
	}

	// Authenticate
	if err = client.Authenticate(); err != nil {
		log.Printf("✗ Authentication failed: %v", err)
		log.Printf("  Added to retry queue: %s", name)
		dbUpdate.Retry[hash] = entry
		if saveErr := SaveJSONDatabase(config.JSONPath, dbUpdate, &config); saveErr != nil {
			log.Printf("Warning: Failed to save database: %v", saveErr)
		}
		return fmt.Errorf("authentication failed: %w", err)
	}
	log.Println("Authenticated with Deluge")

	// Connect to daemon
	if err := client.Connect(); err != nil {
		log.Printf("✗ Connection failed: %v", err)
		log.Printf("  Added to retry queue: %s", name)
		dbUpdate.Retry[hash] = entry
		if saveErr := SaveJSONDatabase(config.JSONPath, dbUpdate, &config); saveErr != nil {
			log.Printf("Warning: Failed to save database: %v", saveErr)
		}
		return fmt.Errorf("connection failed: %w", err)
	}
	log.Println("Connected to Deluge daemon")

	// Add magnet
	err = client.AddMagnet(magnetURI, config.DelugeLabel)

	if err != nil {
		// Check if it's a duplicate error
		if strings.Contains(err.Error(), "already in session") {
			log.Printf("⚠ Duplicate (already in Deluge): %s", name)
			// Add to added section
			dbUpdate.Added[hash] = entry
		} else {
			log.Printf("✗ Failed to add: %v", err)
			log.Printf("  Added to retry queue")
			// Add to retry section
			dbUpdate.Retry[hash] = entry
		}
	} else {
		log.Printf("✓ Successfully added to Deluge: %s", name)
		// Add to added section
		dbUpdate.Added[hash] = entry
	}

	// Save to database
	if err := SaveJSONDatabase(config.JSONPath, dbUpdate, &config); err != nil {
		log.Printf("Warning: Failed to save database: %v", err)
	}

	// Reload to show current retry count
	db, _ = LoadJSONDatabase(config.JSONPath)
	log.Printf("Retry queue: %d items", len(db.Retry))

	return nil
}

// SyncWithDeluge syncs database with Deluge, removing entries no longer in Deluge
func SyncWithDeluge(config Config, dryRun bool) error {
	log.Println("Syncing database with Deluge...")

	// Create Deluge client
	client := NewDelugeClient(config.DelugeHost, config.DelugePort, config.DelugePassword)

	// Authenticate
	if err := client.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	log.Println("Authenticated with Deluge")

	// Connect to daemon
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	log.Println("Connected to Deluge daemon")

	// Get torrents by label
	log.Printf("Fetching torrents with label: %s", config.DelugeLabel)
	torrents, err := client.GetTorrentsByLabel(config.DelugeLabel)
	if err != nil {
		return fmt.Errorf("failed to get torrents: %w", err)
	}

	log.Printf("Found %d torrents in Deluge", len(torrents))

	// Load existing database
	db, err := LoadJSONDatabase(config.JSONPath)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	log.Printf("Database has %d added, %d retry", len(db.Added), len(db.Retry))

	// Find entries in database that are NOT in Deluge
	orphaned := []string{}
	for hash := range db.Added {
		if _, exists := torrents[hash]; !exists {
			orphaned = append(orphaned, hash)
		}
	}

	log.Println(strings.Repeat("=", 60))
	log.Println("Sync Results:")
	log.Printf("  In Deluge: %d", len(torrents))
	log.Printf("  In database: %d", len(db.Added)+len(db.Retry))
	log.Printf("  Orphaned (in DB but not Deluge): %d", len(orphaned))
	log.Println(strings.Repeat("=", 60))

	if len(orphaned) > 0 {
		if dryRun {
			log.Println("\nDry run - would remove:")
			for i, hash := range orphaned {
				if i < 10 || i >= len(orphaned)-10 {
					entry := db.Added[hash]
					log.Printf("  %s - %s", hash[:8], entry.Title)
				} else if i == 10 {
					log.Printf("  ... (%d more) ...", len(orphaned)-20)
				}
			}
			log.Println("\nRun with --sync to actually remove orphaned entries")
		} else {
			log.Printf("\nRemoving %d orphaned entries...", len(orphaned))
			for _, hash := range orphaned {
				delete(db.Added, hash)
			}

			// Save updated database
			homeDir, _ := getHomeDir()
			localPath := filepath.Join(homeDir, "magnet-list-local.json")

			if err := SaveDatabaseLocal(localPath, db); err != nil {
				return fmt.Errorf("failed to save: %w", err)
			}
			log.Printf("Saved to local: %s", localPath)

			// Sync to remote
			remotePath := GetRemotePath(&config)
			if remotePath != "" {
				if err := SaveDatabaseLocal(remotePath, db); err != nil {
					log.Printf("Warning: Could not sync to remote: %v", err)
				} else {
					log.Printf("Synced to remote: %s", remotePath)
				}
			}

			log.Printf("\n✓ Removed %d orphaned entries", len(orphaned))
			log.Printf("Database now has %d entries", len(db.Added)+len(db.Retry))
		}
	} else {
		log.Println("\n✓ Database is in sync with Deluge")
	}

	return nil
}

// BackfillFromDeluge backfills database from existing Deluge torrents
func BackfillFromDeluge(config Config) error {
	log.Println("Backfilling database from Deluge...")

	// Create Deluge client
	client := NewDelugeClient(config.DelugeHost, config.DelugePort, config.DelugePassword)

	// Authenticate
	if err := client.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	log.Println("Authenticated with Deluge")

	// Connect to daemon
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	log.Println("Connected to Deluge daemon")

	// Get torrents by label
	log.Printf("Fetching torrents with label: %s", config.DelugeLabel)
	torrents, err := client.GetTorrentsByLabel(config.DelugeLabel)
	if err != nil {
		return fmt.Errorf("failed to get torrents: %w", err)
	}

	log.Printf("Found %d torrents in Deluge", len(torrents))

	// Load existing database with smart sync (will merge local + remote)
	db, err := LoadJSONDatabase(config.JSONPath)
	if err != nil {
		log.Printf("Warning: Could not load database: %v", err)
		db = &MagnetDatabase{
			Metadata: DatabaseMetadata{},
			Added:    make(map[string]MagnetEntry),
			Retry:    make(map[string]MagnetEntry),
		}
	}

	log.Printf("Loaded existing database: %d added, %d retry, last_sequence=%d",
		len(db.Added), len(db.Retry), db.Metadata.LastSequence)

	// Add torrents to database
	added := 0
	skipped := 0
	nextID := db.Metadata.LastSequence + 1

	for hash, torrentData := range torrents {
		// Check if already exists
		if _, exists := db.Added[hash]; exists {
			skipped++
			continue
		}
		if _, exists := db.Retry[hash]; exists {
			// Move from retry to added
			entry := db.Retry[hash]
			delete(db.Retry, hash)
			db.Added[hash] = entry
			log.Printf("Moved from retry to added: %s", entry.Title)
			added++
			continue
		}

		// Create new entry
		name, _ := torrentData["name"].(string)
		savePath, _ := torrentData["save_path"].(string)

		entry := MagnetEntry{
			UUID:        GenerateUUID(),
			ID:          nextID,
			Title:       name,
			Hash:        hash,
			URI:         fmt.Sprintf("magnet:?xt=urn:btih:%s", hash),
			AddedDate:   time.Now().Format(time.RFC3339),
			SavePath:    savePath,
			TorrentName: name,
		}

		db.Added[hash] = entry
		nextID++
		added++

		if added%100 == 0 {
			log.Printf("Processed %d torrents...", added+skipped)
		}
	}

	db.Metadata.LastSequence = nextID - 1

	// Always save to local first (fast, reliable)
	homeDir, _ := getHomeDir()
	localPath := filepath.Join(homeDir, "magnet-list-local.json")

	if err := SaveDatabaseLocal(localPath, db); err != nil {
		return fmt.Errorf("failed to save to local: %w", err)
	}
	log.Printf("Saved to local: %s", localPath)

	// Try to sync to remote (best effort)
	remotePath := GetRemotePath(&config)
	if remotePath != "" {
		if err := SaveDatabaseLocal(remotePath, db); err != nil {
			log.Printf("Warning: Could not sync to remote %s: %v", remotePath, err)
			log.Println("Changes saved locally, will sync on next operation")
		} else {
			log.Printf("Synced to remote: %s", remotePath)
		}
	}

	// Check for duplicate IDs
	idMap := make(map[int64][]string)
	for hash, entry := range db.Added {
		idMap[entry.ID] = append(idMap[entry.ID], hash)
	}
	for hash, entry := range db.Retry {
		idMap[entry.ID] = append(idMap[entry.ID], hash)
	}

	duplicateIDs := 0
	for id, hashes := range idMap {
		if len(hashes) > 1 {
			duplicateIDs++
			log.Printf("WARNING: ID %d is used by %d entries: %v", id, len(hashes), hashes)
		}
	}

	log.Println(strings.Repeat("=", 60))
	log.Println("Backfill Summary:")
	log.Printf("  Torrents processed: %d", added+skipped)
	log.Printf("    New entries added: %d", added)
	log.Printf("    Already tracked: %d", skipped)
	log.Printf("  Total in database: %d (added: %d, retry: %d)", len(db.Added)+len(db.Retry), len(db.Added), len(db.Retry))
	log.Printf("  Last sequence ID: %d", db.Metadata.LastSequence)
	if duplicateIDs > 0 {
		log.Printf("  ⚠ WARNING: %d duplicate IDs found!", duplicateIDs)
	}
	log.Println(strings.Repeat("=", 60))

	return nil
}

// ProcessRetryQueue processes all items in the retry queue
func ProcessRetryQueue(config Config) error {
	log.Println("Processing retry queue...")

	// Load database
	db, err := LoadJSONDatabase(config.JSONPath)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	if len(db.Retry) == 0 {
		log.Println("✓ Retry queue is empty")
		return nil
	}

	log.Printf("Found %d items in retry queue", len(db.Retry))

	// Create Deluge client
	client := NewDelugeClient(config.DelugeHost, config.DelugePort, config.DelugePassword)

	// Authenticate
	if err := client.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	log.Println("Authenticated with Deluge")

	// Connect to daemon
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	log.Println("Connected to Deluge daemon")

	// Process each retry item
	success := 0
	duplicate := 0
	failed := 0

	for hash, entry := range db.Retry {
		log.Printf("\nRetrying [%d/%d]: %s (attempt #%d)", success+duplicate+failed+1, len(db.Retry), entry.Title, entry.RetryCount+1)

		err := client.AddMagnet(entry.URI, config.DelugeLabel)

		// Update entry
		entry.LastAttempt = time.Now().Format(time.RFC3339)
		entry.RetryCount++

		dbUpdate := &MagnetDatabase{
			Added: make(map[string]MagnetEntry),
			Retry: make(map[string]MagnetEntry),
		}

		if err != nil {
			if strings.Contains(err.Error(), "already in session") {
				log.Printf("  ⚠ Duplicate (already in Deluge)")
				dbUpdate.Added[hash] = entry
				duplicate++
			} else {
				log.Printf("  ✗ Still failing: %v", err)
				dbUpdate.Retry[hash] = entry
				failed++
			}
		} else {
			log.Printf("  ✓ Success!")
			dbUpdate.Added[hash] = entry
			success++
		}

		// Save after each attempt
		if err := SaveJSONDatabase(config.JSONPath, dbUpdate, &config); err != nil {
			log.Printf("Warning: Failed to save database: %v", err)
		}

		// Small delay between attempts
		time.Sleep(1 * time.Second)
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("Retry Summary:")
	log.Printf("  Successfully added: %d", success)
	log.Printf("  Duplicates: %d", duplicate)
	log.Printf("  Still failing: %d", failed)
	log.Println(strings.Repeat("=", 60))

	return nil
}

func main() {
	registerFlag := flag.Bool("register", false, "Register as magnet protocol handler")
	unregisterFlag := flag.Bool("unregister", false, "Unregister magnet protocol handler")
	retryFlag := flag.Bool("retry", false, "Process all items in retry queue")
	backfillFlag := flag.Bool("backfill", false, "Backfill database from existing Deluge torrents")
	syncFlag := flag.Bool("sync", false, "Remove database entries for torrents no longer in Deluge")
	syncDryRunFlag := flag.Bool("sync-dry-run", false, "Show what would be removed without actually removing")
	migrateFlag := flag.Bool("migrate", false, "Migrate JSON files to new format with proper checksums")
	versionFlag := flag.Bool("version", false, "Show version")

	// Configuration flags
	delugeHostFlag := flag.String("host", "", "Deluge server host (e.g., 192.168.1.100)")
	delugePortFlag := flag.String("port", "", "Deluge server port (default: 8112)")
	delugePasswordFlag := flag.String("password", "", "Deluge server password")
	delugeLabelFlag := flag.String("label", "", "Deluge label for torrents (e.g., audiobooks)")
	remotePathFlag := flag.String("remote-path", "", "Path to shared/network storage for syncing (e.g., /mnt/nas/magnet-list.json)")
	saveSettingsFlag := flag.Bool("save-settings", false, "Save command-line settings to config file for future use")
	flag.Parse()

	// Setup logging - use platform-specific log directory
	logDir := GetDefaultLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logDir = "."
	}
	logFile := filepath.Join(logDir, fmt.Sprintf("magnet-handler-%d.log", os.Getpid()))
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer f.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}

	if *versionFlag {
		fmt.Printf("magnet-handler version %s\n", version)
		return
	}

	if *registerFlag {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to get executable path: %v", err)
		}
		if err := RegisterProtocolHandler(exePath); err != nil {
			log.Fatalf("Failed to register protocol handler: %v", err)
		}
		return
	}

	if *unregisterFlag {
		if err := UnregisterProtocolHandler(); err != nil {
			log.Fatalf("Failed to unregister protocol handler: %v", err)
		}
		return
	}

	// Load config (needed for retry, backfill, migrate, and magnet handling)
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Warning: Failed to load config, using defaults: %v", err)
		config = DefaultConfig()
	}

	// Apply command-line overrides
	hasOverrides := false
	if *delugeHostFlag != "" {
		config.DelugeHost = *delugeHostFlag
		hasOverrides = true
	}
	if *delugePortFlag != "" {
		config.DelugePort = *delugePortFlag
		hasOverrides = true
	}
	if *delugePasswordFlag != "" {
		config.DelugePassword = *delugePasswordFlag
		hasOverrides = true
	}
	if *delugeLabelFlag != "" {
		config.DelugeLabel = *delugeLabelFlag
		hasOverrides = true
	}
	if *remotePathFlag != "" {
		config.RemotePath = *remotePathFlag
		hasOverrides = true
	}

	// Save settings if requested
	if *saveSettingsFlag {
		if !hasOverrides {
			log.Fatal("Error: --save-settings requires at least one setting flag (--host, --port, --password, --label, or --remote-path)")
		}
		if err := SaveConfig(config); err != nil {
			log.Printf("Warning: Failed to save config: %v", err)
		} else {
			log.Printf("Settings saved to config file:")
			log.Printf("  Host: %s", config.DelugeHost)
			log.Printf("  Port: %s", config.DelugePort)
			log.Printf("  Label: %s", config.DelugeLabel)
			log.Printf("  Remote path: %s", config.RemotePath)
		}
		// If only saving settings (no other operation or magnet URI), exit cleanly
		if len(flag.Args()) == 0 && !*migrateFlag && !*backfillFlag && !*retryFlag && !*syncFlag && !*syncDryRunFlag {
			return
		}
	}

	// Warn if using default IP (likely not correct)
	if config.DelugeHost == "192.168.0.1" && !hasOverrides {
		log.Printf("WARNING: Using default Deluge host (192.168.0.1) - this is probably not correct!")
		log.Printf("         Set your actual Deluge server IP with: --host YOUR_IP --save-settings")
	}

	if *migrateFlag {
		log.Println("Migrating both local and remote databases...")

		// Migrate local
		if err := MigrateFileFormat(config.JSONPath); err != nil {
			log.Printf("Error migrating local: %v", err)
		}

		// Migrate remote
		remotePath := GetRemotePath(&config)
		if remotePath != "" {
			if err := MigrateFileFormat(remotePath); err != nil {
				log.Printf("Error migrating remote: %v", err)
			}
		} else {
			log.Println("No remote path configured, skipping remote migration")
		}

		log.Println("✓ Migration complete")
		return
	}

	if *backfillFlag {
		if err := BackfillFromDeluge(config); err != nil {
			log.Fatalf("Failed to backfill from Deluge: %v", err)
		}
		return
	}

	if *syncDryRunFlag {
		if err := SyncWithDeluge(config, true); err != nil {
			log.Fatalf("Sync dry run failed: %v", err)
		}
		return
	}

	if *syncFlag {
		if err := SyncWithDeluge(config, false); err != nil {
			log.Fatalf("Sync failed: %v", err)
		}
		return
	}

	if *retryFlag {
		if err := ProcessRetryQueue(config); err != nil {
			log.Fatalf("Failed to process retry queue: %v", err)
		}
		return
	}

	// Handle magnet URI
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("No magnet URI provided")
	}

	magnetURI := args[0]
	// Clean up URI (remove quotes that may be added by shell)
	magnetURI = strings.Trim(magnetURI, `"'`)

	// Process magnet
	if err := AddMagnetToDeluge(magnetURI, config); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
