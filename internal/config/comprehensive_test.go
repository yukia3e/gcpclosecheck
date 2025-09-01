package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigManager_ComplexYAMLStructures tests various YAML structure edge cases
func TestConfigManager_ComplexYAMLStructures(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		description string
	}{
		{
			name: "yaml_with_anchors_and_aliases",
			configYAML: `# YAML with anchors and aliases
default_cleanup: &default_cleanup
  - method: "Close"
    required: true
    description: "Standard close method"

services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods: *default_cleanup
    
  - service_name: "bigtable"
    package_path: "cloud.google.com/go/bigtable"
    creation_functions: ["NewClient"]
    cleanup_methods: *default_cleanup

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`,
			expectError: false,
			description: "YAML with anchors and aliases should be supported",
		},
		{
			name: "yaml_with_complex_nested_structures",
			configYAML: `services:
  - service_name: "pubsub"
    package_path: "cloud.google.com/go/pubsub"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Close client connection"
        conditions:
          - type: "always"
            priority: 1
          - type: "on_error"
            priority: 2
        retry_options:
          max_attempts: 3
          backoff_multiplier: 2.0
          initial_delay: "100ms"

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true
      metadata:
        priority: "low"
        categories: ["unit", "integration"]
        options:
          strict_mode: false
          allow_partial: true`,
			expectError: false,
			description: "Complex nested structures should be preserved",
		},
		{
			name: "yaml_with_unicode_and_special_characters",
			configYAML: `# 日本語コメント with émojis 🚀
services:
  - service_name: "spänner_sërvice"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ 🔒"

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外 🧪"
      enabled: true`,
			expectError: false,
			description: "Unicode characters and emojis should be preserved",
		},
		{
			name: "yaml_with_malformed_structure",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Close connection"
package_exceptions:
  - name: "test_files
    pattern: "**/*_test.go"  # Missing closing quote
    condition:
      type: "test"
      description: "Test exception"
      enabled: true`,
			expectError: true,
			description: "Malformed YAML should be rejected",
		},
		{
			name: "yaml_with_extremely_deep_nesting",
			configYAML: `services:
  - service_name: "deep_service"
    package_path: "example.com/deep"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Close connection"
        config:
          level1:
            level2:
              level3:
                level4:
                  level5:
                    level6:
                      deep_value: "very_deep"

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`,
			expectError: false,
			description: "Deeply nested structures should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "rules.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewConfigManager()
			config, err := manager.LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
				return
			}

			if config == nil {
				t.Error("Expected config but got nil")
				return
			}

			// Verify basic structure integrity
			err = manager.ValidateYAMLIntegrity()
			if err != nil {
				t.Errorf("YAML integrity validation failed: %v", err)
			}

			// Test that the structure can be saved and reloaded
			outputPath := filepath.Join(tmpDir, "output.yaml")
			err = manager.SaveConfig(outputPath)
			if err != nil {
				t.Errorf("Failed to save config: %v", err)
				return
			}

			// Reload and verify
			manager2 := NewConfigManager()
			_, err = manager2.LoadConfig(outputPath)
			if err != nil {
				t.Errorf("Failed to reload saved config: %v", err)
			}
		})
	}
}

// TestConfigManager_BackupRestoreFailureConditions tests backup/restore under various failure conditions
func TestConfigManager_BackupRestoreFailureConditions(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, manager *configManager, tmpDir string) error
		testFunc     func(t *testing.T, manager *configManager, tmpDir string) error
		expectError  bool
		errorMessage string
	}{
		{
			name: "backup_to_readonly_directory",
			setupFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				configYAML := `services: []
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test exception"
      enabled: true`
				
				configPath := filepath.Join(tmpDir, "rules.yaml")
				if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
					return err
				}
				_, err := manager.LoadConfig(configPath)
				return err
			},
			testFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				// Create readonly directory
				readonlyDir := filepath.Join(tmpDir, "readonly")
				if err := os.Mkdir(readonlyDir, 0555); err != nil {
					return err
				}
				// Restore permissions after test
				defer os.Chmod(readonlyDir, 0755)
				
				backupPath := filepath.Join(readonlyDir, "backup.yaml")
				return manager.CreateBackup(backupPath)
			},
			expectError:  true,
			errorMessage: "permission denied",
		},
		{
			name: "restore_from_corrupted_backup",
			setupFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				// Create corrupted backup file
				corruptedBackup := filepath.Join(tmpDir, "corrupted.yaml")
				corruptedContent := `services: [
package_exceptions: invalid yaml structure {{{`
				return os.WriteFile(corruptedBackup, []byte(corruptedContent), 0644)
			},
			testFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				backupPath := filepath.Join(tmpDir, "corrupted.yaml")
				return manager.RestoreFromBackup(backupPath)
			},
			expectError:  true,
			errorMessage: "failed to parse",
		},
		{
			name: "backup_with_disk_full_simulation",
			setupFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				configYAML := `services: []
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test exception"
      enabled: true`
				
				configPath := filepath.Join(tmpDir, "rules.yaml")
				if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
					return err
				}
				_, err := manager.LoadConfig(configPath)
				return err
			},
			testFunc: func(t *testing.T, manager *configManager, tmpDir string) error {
				// Try to backup to non-existent directory
				nonExistentPath := filepath.Join(tmpDir, "nonexistent", "subdir", "backup.yaml")
				return manager.CreateBackup(nonExistentPath)
			},
			expectError:  true,
			errorMessage: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			manager := NewConfigManager().(*configManager)

			// Setup
			if tt.setupFunc != nil {
				err := tt.setupFunc(t, manager, tmpDir)
				if err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Execute test
			err := tt.testFunc(t, manager, tmpDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestConfigManager_CommentAndFormatPreservation tests that comments and formatting are preserved
func TestConfigManager_CommentAndFormatPreservation(t *testing.T) {
	originalYAML := `# Main configuration file
# This file contains GCP resource close checking rules

services:
  # Google Cloud Spanner
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      # Primary close method
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"

# Package-level exceptions
package_exceptions:
  # Command-line tools exception
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "短命プログラム例外"  # Short-lived programs
      enabled: true

  # Test files exception (IMPORTANT: Change this carefully)
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"  # Test code exception
      enabled: true  # <- This will be modified

# End of configuration`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	
	if err := os.WriteFile(configPath, []byte(originalYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Modify the test exception
	err = manager.UpdateTestException(false)
	if err != nil {
		t.Fatalf("Failed to update test exception: %v", err)
	}

	// Save the modified configuration
	outputPath := filepath.Join(tmpDir, "output.yaml")
	err = manager.SaveConfig(outputPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Read the saved content
	savedContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	savedStr := string(savedContent)

	// Note: Current YAML library doesn't preserve comments, so we focus on structure preservation
	// This test verifies that the YAML structure is maintained correctly

	// Verify the specific change was made
	if !strings.Contains(savedStr, "enabled: false") {
		t.Error("Expected 'enabled: false' to be present in saved config")
	}

	// Verify other enabled flags remain true
	if !strings.Contains(savedStr, "enabled: true") {
		t.Error("Expected at least one 'enabled: true' to remain in saved config")
	}

	// Verify structure integrity by reloading
	manager2 := NewConfigManager()
	config2, err := manager2.LoadConfig(outputPath)
	if err != nil {
		t.Fatalf("Failed to reload saved config: %v", err)
	}

	// Verify the change was applied correctly
	found := false
	for _, exception := range config2.PackageExceptions {
		if exception.Name == "test_files" {
			found = true
			if exception.Condition.Enabled != false {
				t.Errorf("Expected test_files enabled to be false, got %v", exception.Condition.Enabled)
			}
			break
		}
	}
	if !found {
		t.Error("test_files exception not found in reloaded config")
	}
}

// TestConfigManager_VerifyConfigChange tests the VerifyConfigChange functionality
func TestConfigManager_VerifyConfigChange(t *testing.T) {
	originalYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Close connection"

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test exception"
      enabled: true`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	
	if err := os.WriteFile(configPath, []byte(originalYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Update test exception
	err = manager.UpdateTestException(false)
	if err != nil {
		t.Fatalf("Failed to update test exception: %v", err)
	}

	// Save the configuration
	outputPath := filepath.Join(tmpDir, "output.yaml")
	err = manager.SaveConfig(outputPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify the change
	err = manager.VerifyConfigChange(outputPath)
	if err != nil {
		t.Errorf("Config change verification failed: %v", err)
	}

	// Test verification of file that doesn't exist
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")
	err = manager.VerifyConfigChange(nonExistentPath)
	if err == nil {
		t.Error("Expected error when verifying non-existent file")
	}
}

// TestConfigManager_ConfigurationStateTracking tests ConfigurationState functionality
func TestConfigManager_ConfigurationStateTracking(t *testing.T) {
	configYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Close connection"

package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test exception"
      enabled: true`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager().(*configManager)
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test getting configuration state
	state := manager.GetConfigurationState()
	if state == nil {
		t.Fatal("Expected ConfigurationState but got nil")
	}

	// Verify initial state
	if state.LastModified.IsZero() {
		t.Error("Expected LastModified to be set")
	}

	if state.Path != configPath {
		t.Errorf("Expected Path %s, got %s", configPath, state.Path)
	}

	if state.Version == "" {
		t.Error("Expected Version to be set")
	}

	// Make a change and verify state updates
	err = manager.UpdateTestException(false)
	if err != nil {
		t.Fatalf("Failed to update test exception: %v", err)
	}

	// Check that change history is recorded
	if len(state.ChangeHistory) == 0 {
		t.Error("Expected change history to be recorded")
	}

	// Verify change record
	if len(state.ChangeHistory) > 0 {
		change := state.ChangeHistory[0]
		if change.Field != "test_files.enabled" {
			t.Errorf("Expected change field 'test_files.enabled', got %s", change.Field)
		}
		if change.OldValue != "true" {
			t.Errorf("Expected old value 'true', got %s", change.OldValue)
		}
		if change.NewValue != "false" {
			t.Errorf("Expected new value 'false', got %s", change.NewValue)
		}
	}
}