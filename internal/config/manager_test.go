package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigManager_LoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid rules.yaml",
			content: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`,
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			content: "invalid: yaml: content:",
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "rules.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewConfigManager()
			config, err := manager.LoadConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if config == nil {
				t.Error("Expected config but got nil")
			}

			// Verify specific fields for valid config
			if !tt.wantErr && tt.name == "valid rules.yaml" {
				if len(config.Services) == 0 {
					t.Error("Expected services to be loaded")
				}
				if len(config.PackageExceptions) == 0 {
					t.Error("Expected package exceptions to be loaded")
				}
			}
		})
	}
}

func TestConfigManager_ValidateYAMLIntegrity(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
	}{
		{
			name: "valid config loaded from file",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`,
			wantErr: false,
		},
		{
			name: "config with missing required fields",
			configYAML: `services: []
package_exceptions: []`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "rules.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewConfigManager()
			_, err := manager.LoadConfig(configPath)
			if err != nil && !tt.wantErr {
				t.Fatalf("Failed to load config: %v", err)
			}

			err = manager.ValidateYAMLIntegrity()

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConfigManager_ValidateYAMLIntegrity_NoConfigLoaded(t *testing.T) {
	manager := NewConfigManager()
	err := manager.ValidateYAMLIntegrity()
	
	if err == nil {
		t.Error("Expected error when no config is loaded")
	}
	
	expected := "no configuration loaded"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestConfigManager_SaveConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
	}{
		{
			name: "valid config save",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file for loading
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "rules.yaml")
			outputPath := filepath.Join(tmpDir, "output.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewConfigManager()
			_, err := manager.LoadConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			err = manager.SaveConfig(outputPath)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify file was created
			if !tt.wantErr {
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Error("Expected output file to be created")
				}

				// Verify content can be loaded back
				manager2 := NewConfigManager()
				_, err := manager2.LoadConfig(outputPath)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
				}
			}
		})
	}
}

func TestConfigManager_SaveConfig_NoConfigLoaded(t *testing.T) {
	manager := NewConfigManager()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.yaml")
	
	err := manager.SaveConfig(outputPath)
	
	if err == nil {
		t.Error("Expected error when no config is loaded")
	}
	
	expected := "no configuration to save"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestConfigManager_UpdateTestException(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		wantErr    bool
	}{
		{
			name:    "enable test exception",
			enabled: true,
			wantErr: false,
		},
		{
			name:    "disable test exception",
			enabled: false,
			wantErr: false,
		},
	}

	configYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "rules.yaml")
			
			if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewConfigManager()
			config, err := manager.LoadConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Verify initial state
			found := false
			for _, exception := range config.PackageExceptions {
				if exception.Name == "test_files" {
					found = true
					if exception.Condition.Enabled != true {
						t.Error("Expected initial enabled state to be true")
					}
					break
				}
			}
			if !found {
				t.Fatal("test_files exception not found in config")
			}

			err = manager.UpdateTestException(tt.enabled)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify the change was made (load config again to check)
			if !tt.wantErr {
				outputPath := filepath.Join(tmpDir, "output.yaml")
				if err := manager.SaveConfig(outputPath); err != nil {
					t.Fatalf("Failed to save config: %v", err)
				}

				manager2 := NewConfigManager()
				config2, err := manager2.LoadConfig(outputPath)
				if err != nil {
					t.Fatalf("Failed to reload config: %v", err)
				}

				found := false
				for _, exception := range config2.PackageExceptions {
					if exception.Name == "test_files" {
						found = true
						if exception.Condition.Enabled != tt.enabled {
							t.Errorf("Expected enabled state to be %v, got %v", tt.enabled, exception.Condition.Enabled)
						}
						break
					}
				}
				if !found {
					t.Error("test_files exception not found in saved config")
				}
			}
		})
	}
}

func TestConfigManager_UpdateTestException_NoConfigLoaded(t *testing.T) {
	manager := NewConfigManager()
	err := manager.UpdateTestException(false)
	
	if err == nil {
		t.Error("Expected error when no config is loaded")
	}
	
	expected := "no configuration loaded"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestConfigManager_UpdateTestException_NotFound(t *testing.T) {
	configYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "other_files"
    pattern: "**/*_other.go"
    condition:
      type: "test"
      description: "Other exception"
      enabled: true`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	err = manager.UpdateTestException(false)
	
	if err == nil {
		t.Error("Expected error when test_files exception not found")
	}
	
	expected := "test_files exception not found in configuration"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestConfigManager_CreateBackup(t *testing.T) {
	configYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	backupPath := filepath.Join(tmpDir, "backup.yaml")
	
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	err = manager.CreateBackup(backupPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify backup file was created
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Expected backup file to be created")
	}

	// Verify backup content can be loaded
	manager2 := NewConfigManager()
	_, err = manager2.LoadConfig(backupPath)
	if err != nil {
		t.Errorf("Failed to load backup config: %v", err)
	}
}

func TestConfigManager_CreateBackup_NoConfigLoaded(t *testing.T) {
	manager := NewConfigManager()
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.yaml")
	
	err := manager.CreateBackup(backupPath)
	
	if err == nil {
		t.Error("Expected error when no config is loaded")
	}
	
	expected := "no configuration loaded"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestConfigManager_CompareConfigurations(t *testing.T) {
	configYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`

	modifiedConfigYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: false`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	modifiedPath := filepath.Join(tmpDir, "modified.yaml")
	
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(modifiedPath, []byte(modifiedConfigYAML), 0644); err != nil {
		t.Fatalf("Failed to create modified test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	diffs, err := manager.CompareConfigurations(modifiedPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(diffs) == 0 {
		t.Error("Expected differences but got none")
	}

	// Check that we detected the enabled flag change
	found := false
	for _, diff := range diffs {
		if strings.Contains(diff, "test_files") && strings.Contains(diff, "enabled") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find test_files enabled flag difference")
	}
}

func TestConfigManager_RestoreFromBackup(t *testing.T) {
	originalConfigYAML := `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true`

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.yaml")
	
	if err := os.WriteFile(backupPath, []byte(originalConfigYAML), 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	manager := NewConfigManager()
	err := manager.RestoreFromBackup(backupPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the configuration was loaded properly
	err = manager.ValidateYAMLIntegrity()
	if err != nil {
		t.Errorf("Configuration validation failed after restore: %v", err)
	}
}

func TestConfigManager_UpdateTestException_PreserveStructure(t *testing.T) {
	// Test with a more complex YAML structure that includes comments
	configYAML := `# Configuration file for gcpclosecheck
services:
  # Cloud Spanner
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クライアント接続のクローズ"

package_exceptions:
  # Short-lived program exception
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "短命プログラム例外"
      enabled: true

  # Test code exception (default disabled)
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true  # This should change to false`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")
	
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewConfigManager()
	_, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Change test exception to disabled
	err = manager.UpdateTestException(false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Save the configuration to verify structure preservation
	outputPath := filepath.Join(tmpDir, "output.yaml")
	err = manager.SaveConfig(outputPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Read the saved file as text and verify it contains expected structure
	savedContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	savedStr := string(savedContent)

	// Verify test_files enabled was changed to false
	if !strings.Contains(savedStr, "enabled: false") {
		t.Error("Expected saved config to contain 'enabled: false'")
	}

	// Verify other enabled flags are preserved as true
	cmdEnabledCount := strings.Count(savedStr, "enabled: true")
	if cmdEnabledCount == 0 {
		t.Error("Expected at least one 'enabled: true' to be preserved")
	}

	// Load the saved config to verify it's still valid
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
		}
	}
	if !found {
		t.Error("test_files exception not found in saved config")
	}
}

func TestConfigManager_UpdateTestException_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name           string
		configYAML     string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "multiple_test_exceptions",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クローズ"
package_exceptions:
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "テストコード例外"
      enabled: true
  - name: "test_files"  # Duplicate name
    pattern: "**/*_spec.go"
    condition:
      type: "test"
      description: "スペックファイル例外"
      enabled: true`,
			expectError: false, // Should update the first occurrence
		},
		{
			name: "no_package_exceptions",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クローズ"`,
			expectError:    true,
			expectedErrMsg: "test_files exception not found in configuration",
		},
		{
			name: "empty_package_exceptions",
			configYAML: `services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "クローズ"
package_exceptions: []`,
			expectError:    true,
			expectedErrMsg: "test_files exception not found in configuration",
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
			_, err := manager.LoadConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			err = manager.UpdateTestException(false)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}