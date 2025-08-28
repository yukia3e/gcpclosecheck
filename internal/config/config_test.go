package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create test configuration file
	testYAML := `
services:
  - service_name: "spanner"
    package_path: "cloud.google.com/go/spanner"
    creation_functions:
      - "NewClient"
      - "ReadOnlyTransaction"
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Client connection close"
      - method: "Stop"
        required: true
        description: "RowIterator stop"
  - service_name: "storage"
    package_path: "cloud.google.com/go/storage"
    creation_functions:
      - "NewClient"
      - "NewReader"
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Stream connection close"
`

	// Create temporary file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("Failed to create test configuration file: %v", err)
	}

	// Test configuration loading
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	if config == nil {
		t.Fatal("Configuration is nil")
	}

	if len(config.Services) != 2 {
		t.Errorf("Service count mismatch: got %d, want 2", len(config.Services))
	}

	// Verify Spanner service
	spannerService := findServiceByName(config.Services, "spanner")
	if spannerService == nil {
		t.Fatal("Spanner service not found")
	}
	if spannerService.PackagePath != "cloud.google.com/go/spanner" {
		t.Errorf("Spanner package path mismatch: %s", spannerService.PackagePath)
	}
	if len(spannerService.CreationFuncs) != 2 {
		t.Errorf("Spanner creation function count mismatch: %d", len(spannerService.CreationFuncs))
	}
	if len(spannerService.CleanupMethods) != 2 {
		t.Errorf("Spanner cleanup method count mismatch: %d", len(spannerService.CleanupMethods))
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	// Test default configuration loading
	config, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("Failed to load default configuration: %v", err)
	}

	if config == nil {
		t.Fatal("Default configuration is nil")
	}

	// Check if minimum required services are included
	expectedServices := []string{"spanner", "storage", "pubsub", "vision"}
	for _, serviceName := range expectedServices {
		if findServiceByName(config.Services, serviceName) == nil {
			t.Errorf("Expected service %s not found", serviceName)
		}
	}

	// Check if package exceptions are configured
	if len(config.PackageExceptions) < 3 {
		t.Errorf("Insufficient package exception count: got %d, want >= 3", len(config.PackageExceptions))
	}

	// Verify default exception existence
	expectedExceptions := []string{"cmd_short_lived", "cloud_functions", "test_files"}
	for _, exceptionName := range expectedExceptions {
		if findExceptionByName(config.PackageExceptions, exceptionName) == nil {
			t.Errorf("Expected exception %s not found", exceptionName)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	// Test invalid configuration
	invalidYAML := `
services:
  - service_name: ""
    package_path: "invalid"
    creation_functions: []
    cleanup_methods: []
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid_config.yaml")
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create invalid test configuration file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Validation test
	if err := config.Validate(); err == nil {
		t.Error("Invalid configuration should result in validation error")
	}
}

func TestConfigGetService(t *testing.T) {
	// Test GetService method
	testYAML := `
services:
  - service_name: "test_service"
    package_path: "test/package"
    creation_functions: ["TestFunc"]
    cleanup_methods:
      - method: "TestClose"
        required: true
        description: "Test close"
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_service_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("Failed to create test configuration file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Get existing service
	service := config.GetService("test_service")
	if service == nil {
		t.Error("Cannot retrieve existing service")
	}

	// Get non-existent service
	nonExistentService := config.GetService("non_existent")
	if nonExistentService != nil {
		t.Error("Non-existent service was retrieved")
	}
}

// TestPackageExceptionRule tests PackageExceptionRule struct behavior
func TestPackageExceptionRule(t *testing.T) {
	testYAML := `
services:
  - service_name: "pubsub"
    package_path: "cloud.google.com/go/pubsub"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Client connection close"

package_exceptions:
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "Short-lived program exception"
      enabled: true
  - name: "cloud_functions"
    pattern: "**/function/**"
    condition:
      type: "cloud_function"
      description: "Cloud Functions exception"
      enabled: true
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test code exception"
      enabled: false
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "exception_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("Failed to create test configuration file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if PackageExceptions are loaded correctly
	if len(config.PackageExceptions) != 3 {
		t.Errorf("Package exception count mismatch: got %d, want 3", len(config.PackageExceptions))
	}

	// Verify cmd_short_lived exception
	cmdException := findExceptionByName(config.PackageExceptions, "cmd_short_lived")
	if cmdException == nil {
		t.Fatal("cmd_short_lived exception not found")
	}
	if cmdException.Pattern != "*/cmd/*" {
		t.Errorf("cmd_short_lived pattern mismatch: got %s, want */cmd/*", cmdException.Pattern)
	}
	if cmdException.Condition.Type != "short_lived" {
		t.Errorf("cmd_short_lived type mismatch: got %s, want short_lived", cmdException.Condition.Type)
	}
	if !cmdException.Condition.Enabled {
		t.Error("cmd_short_lived is disabled")
	}

	// Verify cloud_functions exception
	functionException := findExceptionByName(config.PackageExceptions, "cloud_functions")
	if functionException == nil {
		t.Fatal("cloud_functions exception not found")
	}
	if functionException.Pattern != "**/function/**" {
		t.Errorf("cloud_functions pattern mismatch: got %s, want **/function/**", functionException.Pattern)
	}
	if functionException.Condition.Type != "cloud_function" {
		t.Errorf("cloud_functions type mismatch: got %s, want cloud_function", functionException.Condition.Type)
	}

	// Verify test_files exception (disabled by default)
	testException := findExceptionByName(config.PackageExceptions, "test_files")
	if testException == nil {
		t.Fatal("test_files exception not found")
	}
	if testException.Pattern != "**/*_test.go" {
		t.Errorf("test_files pattern mismatch: got %s, want **/*_test.go", testException.Pattern)
	}
	if testException.Condition.Type != "test" {
		t.Errorf("test_files type mismatch: got %s, want test", testException.Condition.Type)
	}
	if testException.Condition.Enabled {
		t.Error("test_files is enabled by default")
	}
}

// TestPackageExceptionRuleValidation tests PackageExceptionRule validation
func TestPackageExceptionRuleValidation(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_exception",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Test"

package_exceptions:
  - name: "valid"
    pattern: "*/test/*"
    condition:
      type: "short_lived"
      description: "Test exception"
      enabled: true
`,
			expectError: false,
		},
		{
			name: "empty_exception_name",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Test"

package_exceptions:
  - name: ""
    pattern: "*/test/*"
    condition:
      type: "short_lived"
      description: "Test exception"
      enabled: true
`,
			expectError: true,
			errorMsg:    "exception name is empty",
		},
		{
			name: "empty_pattern",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Test"

package_exceptions:
  - name: "test"
    pattern: ""
    condition:
      type: "short_lived"
      description: "Test exception"
      enabled: true
`,
			expectError: true,
			errorMsg:    "pattern is empty",
		},
		{
			name: "invalid_condition_type",
			yaml: `
services:
  - service_name: "test"
    package_path: "test"
    creation_functions: ["Test"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Test"

package_exceptions:
  - name: "test"
    pattern: "*/test/*"
    condition:
      type: "invalid_type"
      description: "Test exception"
      enabled: true
`,
			expectError: true,
			errorMsg:    "invalid condition type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test_config.yaml")
			if err := os.WriteFile(configFile, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("Failed to create test configuration file: %v", err)
			}

			config, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("Failed to load configuration: %v", err)
			}

			err = config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but no error occurred")
				} else if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message not found: got %v, want contains %s", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validation error occurred: %v", err)
				}
			}
		})
	}
}

// TestShouldExemptPackage tests ShouldExemptPackage method
func TestShouldExemptPackage(t *testing.T) {
	testYAML := `
services:
  - service_name: "pubsub"
    package_path: "cloud.google.com/go/pubsub"
    creation_functions: ["NewClient"]
    cleanup_methods:
      - method: "Close"
        required: true
        description: "Client connection close"

package_exceptions:
  - name: "cmd_short_lived"
    pattern: "*/cmd/*"
    condition:
      type: "short_lived"
      description: "Short-lived program exception"
      enabled: true
  - name: "cloud_functions"
    pattern: "**/function/**"
    condition:
      type: "cloud_function"
      description: "Cloud Functions exception"
      enabled: true
  - name: "test_files"
    pattern: "**/*_test.go"
    condition:
      type: "test"
      description: "Test code exception"
      enabled: false
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "exempt_config.yaml")
	if err := os.WriteFile(configFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("Failed to create test configuration file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	tests := []struct {
		name         string
		packagePath  string
		shouldExempt bool
		exemptReason string
	}{
		{
			name:         "cmd_package_should_exempt",
			packagePath:  "github.com/example/project/cmd/server",
			shouldExempt: true,
			exemptReason: "Short-lived program exception",
		},
		{
			name:         "function_package_should_exempt",
			packagePath:  "github.com/example/project/internal/function/handler",
			shouldExempt: true,
			exemptReason: "Cloud Functions exception",
		},
		{
			name:         "test_file_should_not_exempt_when_disabled",
			packagePath:  "github.com/example/project/internal/handler_test.go",
			shouldExempt: false,
			exemptReason: "",
		},
		{
			name:         "regular_package_should_not_exempt",
			packagePath:  "github.com/example/project/internal/handler",
			shouldExempt: false,
			exemptReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldExempt, reason := config.ShouldExemptPackage(tt.packagePath)
			if shouldExempt != tt.shouldExempt {
				t.Errorf("ShouldExemptPackage(%s): got %v, want %v", tt.packagePath, shouldExempt, tt.shouldExempt)
			}
			if reason != tt.exemptReason {
				t.Errorf("ShouldExemptPackage(%s): reason got %s, want %s", tt.packagePath, reason, tt.exemptReason)
			}
		})
	}
}

// Helper function: Find service by name
func findServiceByName(services []ServiceRule, name string) *ServiceRule {
	for i := range services {
		if services[i].ServiceName == name {
			return &services[i]
		}
	}
	return nil
}

// Helper function: Find package exception by name
func findExceptionByName(exceptions []PackageExceptionRule, name string) *PackageExceptionRule {
	for i := range exceptions {
		if exceptions[i].Name == name {
			return &exceptions[i]
		}
	}
	return nil
}

// TestMessagesIntegration verifies that config package uses messages constants
func TestMessagesIntegration(t *testing.T) {
	// Test LoadConfig with empty path - should use ConfigFileEmpty message
	_, err := LoadConfig("")
	if err == nil {
		t.Error("Expected error for empty config path")
	}
	if !strings.Contains(err.Error(), "Configuration file path is empty") {
		t.Errorf("Expected English error message from messages package, got: %s", err.Error())
	}
	
	// Test LoadConfig with non-existent file - should use ConfigLoadFailed message
	_, err = LoadConfig("/non/existent/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
	if !strings.Contains(err.Error(), "Failed to load configuration file") {
		t.Errorf("Expected English error message from messages package, got: %s", err.Error())
	}
	
	// Test invalid YAML - should use ConfigYAMLParseFailed message
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}
	
	_, err = LoadConfig(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "Failed to parse YAML configuration") {
		t.Errorf("Expected English error message from messages package, got: %s", err.Error())
	}
}

// TestConfigValidationMessagesIntegration tests validation error messages
func TestConfigValidationMessagesIntegration(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		expectedMsg  string
	}{
		{
			name:        "empty_services",
			config:      Config{Services: []ServiceRule{}},
			expectedMsg: "Services definition is empty",
		},
		{
			name: "empty_service_name",
			config: Config{
				Services: []ServiceRule{
					{ServiceName: "", PackagePath: "test", CreationFuncs: []string{"test"}, CleanupMethods: []CleanupMethod{{Method: "Close", Required: true}}},
				},
			},
			expectedMsg: "Service[0]: service name is empty",
		},
		{
			name: "empty_package_path",
			config: Config{
				Services: []ServiceRule{
					{ServiceName: "test", PackagePath: "", CreationFuncs: []string{"test"}, CleanupMethods: []CleanupMethod{{Method: "Close", Required: true}}},
				},
			},
			expectedMsg: "Service[0](test): package path is empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Error("Expected validation error")
			}
			if !strings.Contains(err.Error(), tt.expectedMsg) {
				t.Errorf("Expected English error message %q, got: %s", tt.expectedMsg, err.Error())
			}
		})
	}
}

// Helper function: Check if string contains substring
func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(strings.Contains(s, substr))))
}
