package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigManager manages YAML configuration loading, validation, and modification
type ConfigManager interface {
	// LoadConfig loads configuration from the specified file path
	LoadConfig(path string) (*Config, error)
	
	// ValidateYAMLIntegrity validates the YAML structure and configuration integrity
	ValidateYAMLIntegrity() error
	
	// SaveConfig saves the current configuration to the specified file path
	SaveConfig(path string) error
	
	// UpdateTestException updates the test exception enabled flag
	UpdateTestException(enabled bool) error
	
	// CreateBackup creates a backup of the current configuration
	CreateBackup(backupPath string) error
	
	// CompareConfigurations compares two configurations and returns differences
	CompareConfigurations(otherPath string) ([]string, error)
	
	// RestoreFromBackup restores configuration from a backup file
	RestoreFromBackup(backupPath string) error
	
	// VerifyConfigChange verifies that configuration changes were applied correctly
	VerifyConfigChange(filePath string) error
}

// ChangeRecord represents a single configuration change
type ChangeRecord struct {
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	Timestamp time.Time `json:"timestamp"`
}

// ConfigurationState tracks the current state of the configuration
type ConfigurationState struct {
	Path          string         `json:"path"`
	LastModified  time.Time      `json:"last_modified"`
	Version       string         `json:"version"`
	ChangeHistory []ChangeRecord `json:"change_history"`
}

// configManager is the concrete implementation of ConfigManager
type configManager struct {
	config *Config
	configPath string
	state  *ConfigurationState
}

// NewConfigManager creates a new instance of ConfigManager
func NewConfigManager() ConfigManager {
	return &configManager{
		state: &ConfigurationState{
			ChangeHistory: []ChangeRecord{},
			Version:       "1.0.0",
		},
	}
}

// LoadConfig loads configuration from the specified file path
func (cm *configManager) LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("configuration file path cannot be empty")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("configuration file is empty")
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	cm.config = &config
	cm.configPath = path
	
	// Update state
	if fileInfo, err := os.Stat(path); err == nil {
		cm.state.LastModified = fileInfo.ModTime()
		cm.state.Path = path
	}
	
	return &config, nil
}

// ValidateYAMLIntegrity validates the YAML structure and configuration integrity
func (cm *configManager) ValidateYAMLIntegrity() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Use existing Validate method from config.go
	return cm.config.Validate()
}

// SaveConfig saves the current configuration to the specified file path
func (cm *configManager) SaveConfig(path string) error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateTestException updates the test exception enabled flag
func (cm *configManager) UpdateTestException(enabled bool) error {
	if cm.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Find and update test_files exception
	for i := range cm.config.PackageExceptions {
		if cm.config.PackageExceptions[i].Name == "test_files" {
			oldValue := "true"
			if !cm.config.PackageExceptions[i].Condition.Enabled {
				oldValue = "false"
			}
			newValue := "false"
			if enabled {
				newValue = "true"
			}
			
			cm.config.PackageExceptions[i].Condition.Enabled = enabled
			cm.recordChange("test_files.enabled", oldValue, newValue)
			return nil
		}
	}

	return fmt.Errorf("test_files exception not found in configuration")
}

// CreateBackup creates a backup of the current configuration
func (cm *configManager) CreateBackup(backupPath string) error {
	if cm.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	return cm.SaveConfig(backupPath)
}

// CompareConfigurations compares two configurations and returns differences
func (cm *configManager) CompareConfigurations(otherPath string) ([]string, error) {
	if cm.config == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	// Load the other configuration
	otherData, err := os.ReadFile(filepath.Clean(otherPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read comparison config file: %w", err)
	}

	var otherConfig Config
	if err := yaml.Unmarshal(otherData, &otherConfig); err != nil {
		return nil, fmt.Errorf("failed to parse comparison YAML: %w", err)
	}

	// Compare configurations and return differences
	var diffs []string
	
	// Compare package exceptions specifically
	currentExceptions := make(map[string]*PackageExceptionRule)
	for i := range cm.config.PackageExceptions {
		currentExceptions[cm.config.PackageExceptions[i].Name] = &cm.config.PackageExceptions[i]
	}
	
	otherExceptions := make(map[string]*PackageExceptionRule)
	for i := range otherConfig.PackageExceptions {
		otherExceptions[otherConfig.PackageExceptions[i].Name] = &otherConfig.PackageExceptions[i]
	}
	
	// Check for differences in package exceptions
	for name, currentEx := range currentExceptions {
		otherEx, exists := otherExceptions[name]
		if !exists {
			diffs = append(diffs, fmt.Sprintf("Package exception '%s' removed", name))
			continue
		}
		
		if currentEx.Condition.Enabled != otherEx.Condition.Enabled {
			diffs = append(diffs, fmt.Sprintf("Package exception '%s' enabled changed from %t to %t", 
				name, currentEx.Condition.Enabled, otherEx.Condition.Enabled))
		}
		
		if currentEx.Pattern != otherEx.Pattern {
			diffs = append(diffs, fmt.Sprintf("Package exception '%s' pattern changed from '%s' to '%s'", 
				name, currentEx.Pattern, otherEx.Pattern))
		}
		
		if currentEx.Condition.Description != otherEx.Condition.Description {
			diffs = append(diffs, fmt.Sprintf("Package exception '%s' description changed", name))
		}
	}
	
	// Check for new package exceptions
	for name := range otherExceptions {
		if _, exists := currentExceptions[name]; !exists {
			diffs = append(diffs, fmt.Sprintf("Package exception '%s' added", name))
		}
	}
	
	// Compare services (basic comparison)
	if len(cm.config.Services) != len(otherConfig.Services) {
		diffs = append(diffs, fmt.Sprintf("Services count changed from %d to %d", 
			len(cm.config.Services), len(otherConfig.Services)))
	}
	
	return diffs, nil
}

// RestoreFromBackup restores configuration from a backup file
func (cm *configManager) RestoreFromBackup(backupPath string) error {
	_, err := cm.LoadConfig(backupPath)
	return err
}

// VerifyConfigChange verifies that configuration changes were applied correctly
func (cm *configManager) VerifyConfigChange(filePath string) error {
	if cm.config == nil {
		return fmt.Errorf("no configuration loaded to compare")
	}
	
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Load the file and verify it can be parsed
	tempManager := NewConfigManager().(*configManager)
	_, err := tempManager.LoadConfig(filePath)
	if err != nil {
		return fmt.Errorf("failed to load file for verification: %w", err)
	}
	
	// Verify YAML integrity
	err = tempManager.ValidateYAMLIntegrity()
	if err != nil {
		return fmt.Errorf("YAML integrity validation failed: %w", err)
	}
	
	return nil
}

// GetConfigurationState returns the current configuration state
func (cm *configManager) GetConfigurationState() *ConfigurationState {
	return cm.state
}

// recordChange records a configuration change in the change history
func (cm *configManager) recordChange(field, oldValue, newValue string) {
	if cm.state == nil {
		cm.state = &ConfigurationState{
			ChangeHistory: []ChangeRecord{},
			Version:       "1.0.0",
		}
	}
	
	change := ChangeRecord{
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
	}
	
	cm.state.ChangeHistory = append(cm.state.ChangeHistory, change)
}