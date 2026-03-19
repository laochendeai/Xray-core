package webpanel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ConfigFileManager handles reading, writing, and backing up the Xray config file.
type ConfigFileManager struct {
	configPath string
}

// NewConfigFileManager creates a new ConfigFileManager.
func NewConfigFileManager(configPath string) *ConfigFileManager {
	return &ConfigFileManager{
		configPath: configPath,
	}
}

// ReadConfig reads the current config file.
func (m *ConfigFileManager) ReadConfig() (json.RawMessage, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate JSON
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("config file is not valid JSON: %w", err)
	}

	return raw, nil
}

// WriteConfig writes config data to the config file, creating a backup first.
func (m *ConfigFileManager) WriteConfig(data json.RawMessage) error {
	// Validate JSON
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Create backup before writing
	if _, err := os.Stat(m.configPath); err == nil {
		if err := m.CreateBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Pretty print
	prettyJSON, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	if err := os.WriteFile(m.configPath, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates JSON config without writing it.
func (m *ConfigFileManager) ValidateConfig(data json.RawMessage) error {
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check required fields
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid config structure: %w", err)
	}

	return nil
}

// CreateBackup creates a timestamped backup of the current config.
func (m *ConfigFileManager) CreateBackup() error {
	backupDir := m.getBackupDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("config-%s.json", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	// Keep only the last 20 backups
	m.cleanOldBackups(20)

	return nil
}

// ListBackups returns a list of backup files.
func (m *ConfigFileManager) ListBackups() ([]map[string]interface{}, error) {
	backupDir := m.getBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, map[string]interface{}{
			"name":     entry.Name(),
			"size":     info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		})
	}

	// Sort by name descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i]["name"].(string) > backups[j]["name"].(string)
	})

	return backups, nil
}

// RestoreBackup restores a config from a backup file.
func (m *ConfigFileManager) RestoreBackup(backupName string) error {
	backupPath := filepath.Join(m.getBackupDir(), backupName)

	// Validate the backup name to prevent directory traversal
	if strings.Contains(backupName, "..") || strings.Contains(backupName, "/") {
		return fmt.Errorf("invalid backup name")
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Validate JSON
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("backup is not valid JSON: %w", err)
	}

	// Backup current config before restoring
	if err := m.CreateBackup(); err != nil {
		return fmt.Errorf("failed to backup current config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	return nil
}

func (m *ConfigFileManager) getBackupDir() string {
	return filepath.Join(filepath.Dir(m.configPath), "backups")
}

func (m *ConfigFileManager) cleanOldBackups(maxKeep int) {
	backupDir := m.getBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	var backupFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			backupFiles = append(backupFiles, entry)
		}
	}

	if len(backupFiles) <= maxKeep {
		return
	}

	// Sort by name ascending (oldest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].Name() < backupFiles[j].Name()
	})

	// Remove oldest
	for i := 0; i < len(backupFiles)-maxKeep; i++ {
		os.Remove(filepath.Join(backupDir, backupFiles[i].Name()))
	}
}
