package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yukia3e/gcpclosecheck/internal/config"
	"github.com/yukia3e/gcpclosecheck/internal/issues"
	"github.com/yukia3e/gcpclosecheck/internal/ci"
	"github.com/yukia3e/gcpclosecheck/internal/validation"
)

// TestFixture represents a complete test fixture
type TestFixture struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Config      *config.Config          `json:"config"`
	Files       map[string]string       `json:"files"`
	Expected    *ExpectedResults        `json:"expected"`
	Setup       *SetupInstructions      `json:"setup,omitempty"`
	Metadata    *FixtureMetadata        `json:"metadata"`
}

// ExpectedResults contains expected test outcomes
type ExpectedResults struct {
	Issues         []issues.Issue              `json:"issues,omitempty"`
	CIResults      *ci.CIResult               `json:"ci_results,omitempty"`
	Validation     *validation.ValidationReport `json:"validation,omitempty"`
	ShouldFail     bool                        `json:"should_fail"`
	FailureReasons []string                    `json:"failure_reasons,omitempty"`
}

// SetupInstructions contains setup requirements for the fixture
type SetupInstructions struct {
	RequiredTools    []string          `json:"required_tools,omitempty"`
	EnvironmentVars  map[string]string `json:"environment_vars,omitempty"`
	PreSetupCommands []string          `json:"pre_setup_commands,omitempty"`
	PostSetupCommands []string         `json:"post_setup_commands,omitempty"`
}

// FixtureMetadata contains metadata about the fixture
type FixtureMetadata struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Version      string    `json:"version"`
	Tags         []string  `json:"tags"`
	Category     string    `json:"category"`
	Complexity   string    `json:"complexity"` // simple, medium, complex
	Author       string    `json:"author"`
}

// FixtureManager manages test fixtures and data
type FixtureManager interface {
	// LoadFixture loads a test fixture by name
	LoadFixture(name string) (*TestFixture, error)
	
	// SaveFixture saves a test fixture
	SaveFixture(fixture *TestFixture) error
	
	// ListFixtures returns all available fixtures
	ListFixtures() ([]*TestFixture, error)
	
	// CreateTemporaryEnvironment sets up a temporary test environment
	CreateTemporaryEnvironment(fixture *TestFixture) (string, error)
	
	// CleanupEnvironment cleans up a temporary test environment
	CleanupEnvironment(path string) error
	
	// ValidateFixture validates a fixture's integrity
	ValidateFixture(fixture *TestFixture) error
	
	// GenerateFixtureFromProject generates a fixture from current project state
	GenerateFixtureFromProject(name, description string) (*TestFixture, error)
}

// fixtureManager is the concrete implementation
type fixtureManager struct {
	basePath string
}

// NewFixtureManager creates a new fixture manager
func NewFixtureManager(basePath string) FixtureManager {
	return &fixtureManager{
		basePath: basePath,
	}
}

// TestFixtureManager_LoadFixture tests loading fixtures
func TestFixtureManager_LoadFixture(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	// Create a test fixture file
	fixture := createSampleFixture()
	fixturePath := filepath.Join(tmpDir, "test_fixture.json")
	
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal fixture: %v", err)
	}
	
	if err := os.WriteFile(fixturePath, data, 0644); err != nil {
		t.Fatalf("Failed to write fixture file: %v", err)
	}
	
	// Test loading
	loaded, err := manager.LoadFixture("test_fixture")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}
	
	if loaded == nil {
		t.Fatal("Expected fixture but got nil")
	}
	
	if loaded.Name != fixture.Name {
		t.Errorf("Expected fixture name %q, got %q", fixture.Name, loaded.Name)
	}
	
	if loaded.Description != fixture.Description {
		t.Errorf("Expected fixture description %q, got %q", fixture.Description, loaded.Description)
	}
}

// TestFixtureManager_SaveFixture tests saving fixtures
func TestFixtureManager_SaveFixture(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	fixture := createSampleFixture()
	
	// Test saving
	if err := manager.SaveFixture(fixture); err != nil {
		t.Fatalf("Failed to save fixture: %v", err)
	}
	
	// Verify file was created
	expectedPath := filepath.Join(tmpDir, fixture.Name+".json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected fixture file to be created at %s", expectedPath)
	}
	
	// Verify content
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read saved fixture: %v", err)
	}
	
	var saved TestFixture
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to unmarshal saved fixture: %v", err)
	}
	
	if saved.Name != fixture.Name {
		t.Errorf("Expected saved fixture name %q, got %q", fixture.Name, saved.Name)
	}
}

// TestFixtureManager_ListFixtures tests listing fixtures
func TestFixtureManager_ListFixtures(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	// Create multiple fixtures
	fixtures := []*TestFixture{
		createSampleFixture(),
		createComplexFixture(),
		createErrorFixture(),
	}
	
	for _, fixture := range fixtures {
		if err := manager.SaveFixture(fixture); err != nil {
			t.Fatalf("Failed to save fixture %s: %v", fixture.Name, err)
		}
	}
	
	// Test listing
	listed, err := manager.ListFixtures()
	if err != nil {
		t.Fatalf("Failed to list fixtures: %v", err)
	}
	
	if len(listed) != len(fixtures) {
		t.Errorf("Expected %d fixtures, got %d", len(fixtures), len(listed))
	}
	
	// Verify all fixtures are present
	fixtureNames := make(map[string]bool)
	for _, fixture := range listed {
		fixtureNames[fixture.Name] = true
	}
	
	for _, original := range fixtures {
		if !fixtureNames[original.Name] {
			t.Errorf("Expected fixture %s to be listed", original.Name)
		}
	}
}

// TestFixtureManager_CreateTemporaryEnvironment tests environment creation
func TestFixtureManager_CreateTemporaryEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	fixture := createSampleFixture()
	
	// Test environment creation
	envPath, err := manager.CreateTemporaryEnvironment(fixture)
	if err != nil {
		t.Fatalf("Failed to create temporary environment: %v", err)
	}
	
	if envPath == "" {
		t.Fatal("Expected non-empty environment path")
	}
	
	// Verify environment was created
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf("Expected environment directory to exist at %s", envPath)
	}
	
	// Verify files were created
	for filename := range fixture.Files {
		filePath := filepath.Join(envPath, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist in environment", filename)
		}
	}
	
	// Test cleanup
	if err := manager.CleanupEnvironment(envPath); err != nil {
		t.Errorf("Failed to cleanup environment: %v", err)
	}
}

// TestFixtureManager_ValidateFixture tests fixture validation
func TestFixtureManager_ValidateFixture(t *testing.T) {
	manager := NewFixtureManager(t.TempDir())
	
	tests := []struct {
		name        string
		fixture     *TestFixture
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_fixture",
			fixture:     createSampleFixture(),
			expectError: false,
		},
		{
			name: "missing_name",
			fixture: &TestFixture{
				Description: "Test fixture without name",
				Config:      &config.Config{},
			},
			expectError: true,
			errorMsg:    "name",
		},
		{
			name: "missing_description",
			fixture: &TestFixture{
				Name:   "test_fixture",
				Config: &config.Config{},
			},
			expectError: true,
			errorMsg:    "description",
		},
		{
			name: "nil_config",
			fixture: &TestFixture{
				Name:        "test_fixture",
				Description: "Test fixture with nil config",
				Config:      nil,
			},
			expectError: true,
			errorMsg:    "config",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateFixture(tt.fixture)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error but got none")
				} else if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestFixtureManager_GenerateFixtureFromProject tests fixture generation
func TestFixtureManager_GenerateFixtureFromProject(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	// Create some project files for testing
	projectFiles := map[string]string{
		"main.go": `package main
import "fmt"
func main() {
    fmt.Println("Hello, World!")
}`,
		"config.yaml": `
services:
  - name: "test"
    clients:
      - name: "client"
        close_required: true
`,
	}
	
	for filename, content := range projectFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create project file %s: %v", filename, err)
		}
	}
	
	// Test fixture generation
	fixture, err := manager.GenerateFixtureFromProject("generated_test", "Generated from test project")
	if err != nil {
		t.Fatalf("Failed to generate fixture from project: %v", err)
	}
	
	if fixture == nil {
		t.Fatal("Expected generated fixture but got nil")
	}
	
	if fixture.Name != "generated_test" {
		t.Errorf("Expected fixture name %q, got %q", "generated_test", fixture.Name)
	}
	
	if fixture.Description != "Generated from test project" {
		t.Errorf("Expected fixture description %q, got %q", "Generated from test project", fixture.Description)
	}
	
	// Verify files were captured
	if len(fixture.Files) == 0 {
		t.Error("Expected fixture to capture project files")
	}
}

// TestFixture_Integration tests fixture usage in integration scenarios
func TestFixture_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewFixtureManager(tmpDir)
	
	// Test complete fixture workflow
	fixture := createComplexFixture()
	
	// Save fixture
	if err := manager.SaveFixture(fixture); err != nil {
		t.Fatalf("Failed to save fixture: %v", err)
	}
	
	// Load fixture
	loaded, err := manager.LoadFixture(fixture.Name)
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}
	
	// Validate fixture
	if err := manager.ValidateFixture(loaded); err != nil {
		t.Fatalf("Fixture validation failed: %v", err)
	}
	
	// Create temporary environment
	envPath, err := manager.CreateTemporaryEnvironment(loaded)
	if err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}
	defer manager.CleanupEnvironment(envPath)
	
	// Verify environment contains expected files
	for filename, expectedContent := range loaded.Files {
		filePath := filepath.Join(envPath, filename)
		actualContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", filename, err)
			continue
		}
		
		if string(actualContent) != expectedContent {
			t.Errorf("File %s content mismatch.\nExpected:\n%s\nActual:\n%s", 
				filename, expectedContent, string(actualContent))
		}
	}
}

// createSampleFixture creates a simple test fixture
func createSampleFixture() *TestFixture {
	return &TestFixture{
		Name:        "sample_test",
		Description: "A simple test fixture for basic functionality",
		Config: &config.Config{
			Services: []config.ServiceRule{
				{
					ServiceName:   "test_service",
					PackagePath:   "cloud.google.com/go/test",
					CreationFuncs: []string{"NewClient"},
					CleanupMethods: []config.CleanupMethod{
						{
							Method:      "Close",
							Required:    true,
							Description: "Close the client connection",
						},
					},
				},
			},
		},
		Files: map[string]string{
			"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
			"config.yaml": `services:
  - name: "test_service"
    clients:
      - name: "test_client"
        close_required: true`,
		},
		Expected: &ExpectedResults{
			ShouldFail: false,
		},
		Metadata: &FixtureMetadata{
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    "1.0.0",
			Tags:       []string{"basic", "test"},
			Category:   "functionality",
			Complexity: "simple",
			Author:     "test_system",
		},
	}
}

// createComplexFixture creates a complex test fixture
func createComplexFixture() *TestFixture {
	return &TestFixture{
		Name:        "complex_test",
		Description: "A complex test fixture with multiple components",
		Config: &config.Config{
			Services: []config.ServiceRule{
				{
					ServiceName:   "spanner",
					PackagePath:   "cloud.google.com/go/spanner",
					CreationFuncs: []string{"NewClient"},
					CleanupMethods: []config.CleanupMethod{
						{
							Method:      "Close",
							Required:    true,
							Description: "Close the spanner client",
						},
					},
				},
				{
					ServiceName:   "storage",
					PackagePath:   "cloud.google.com/go/storage",
					CreationFuncs: []string{"NewClient"},
					CleanupMethods: []config.CleanupMethod{
						{
							Method:      "Close",
							Required:    true,
							Description: "Close the storage client",
						},
					},
				},
			},
			PackageExceptions: []config.PackageExceptionRule{
				{
					Name:    "test_files",
					Pattern: "*_test.go",
					Condition: config.ExceptionCondition{
						Enabled:     true,
						Description: "Skip close checks for test files",
					},
				},
			},
		},
		Files: map[string]string{
			"main.go": `package main

import (
	"context"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
)

func main() {
	ctx := context.Background()
	
	// This should trigger a linter error
	spannerClient, _ := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	storageClient, _ := storage.NewClient(ctx)
	
	// Missing close calls - should be detected
	_ = spannerClient
	_ = storageClient
}`,
			"main_test.go": `package main

import "testing"

func TestMain(t *testing.T) {
	// This should be ignored due to test file exception
}`,
		},
		Expected: &ExpectedResults{
			Issues: []issues.Issue{
				{
					File:     "main.go",
					Line:     12,
					Column:   1,
					Linter:   "gcpclosecheck",
					Message:  "spanner client not closed",
					Severity: "error",
				},
				{
					File:     "main.go",
					Line:     13,
					Column:   1,
					Linter:   "gcpclosecheck",
					Message:  "storage client not closed",
					Severity: "error",
				},
			},
			ShouldFail: true,
			FailureReasons: []string{
				"GCP clients not properly closed",
				"Resource leak detected",
			},
		},
		Metadata: &FixtureMetadata{
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    "1.0.0",
			Tags:       []string{"complex", "gcp", "resource_leak"},
			Category:   "resource_management",
			Complexity: "complex",
			Author:     "test_system",
		},
	}
}

// createErrorFixture creates a fixture designed to test error scenarios
func createErrorFixture() *TestFixture {
	return &TestFixture{
		Name:        "error_test",
		Description: "A test fixture for error scenarios",
		Config: &config.Config{
			Services: []config.ServiceRule{
				{
					ServiceName:   "invalid_service",
					PackagePath:   "cloud.google.com/go/invalid",
					CreationFuncs: []string{"NewClient"},
					CleanupMethods: []config.CleanupMethod{
						{
							Method:      "Close",
							Required:    true,
							Description: "Close the client",
						},
					},
				},
			},
		},
		Files: map[string]string{
			"invalid.go": `package main

// This file has syntax errors
func main() {
	// Missing closing brace
	if true {
		fmt.Println("error"
}`,
		},
		Expected: &ExpectedResults{
			ShouldFail: true,
			FailureReasons: []string{
				"Syntax error in source file",
				"Compilation failure",
			},
		},
		Metadata: &FixtureMetadata{
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    "1.0.0",
			Tags:       []string{"error", "syntax", "compilation"},
			Category:   "error_handling",
			Complexity: "medium",
			Author:     "test_system",
		},
	}
}

// containsIgnoreCase checks if a string contains another string (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// LoadFixture loads a test fixture by name
func (fm *fixtureManager) LoadFixture(name string) (*TestFixture, error) {
	filePath := filepath.Join(fm.basePath, name+".json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}
	
	var fixture TestFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture: %w", err)
	}
	
	return &fixture, nil
}

// SaveFixture saves a test fixture
func (fm *fixtureManager) SaveFixture(fixture *TestFixture) error {
	if err := fm.ValidateFixture(fixture); err != nil {
		return fmt.Errorf("fixture validation failed: %w", err)
	}
	
	// Update metadata
	if fixture.Metadata == nil {
		fixture.Metadata = &FixtureMetadata{
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		}
	}
	fixture.Metadata.UpdatedAt = time.Now()
	
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal fixture: %w", err)
	}
	
	filePath := filepath.Join(fm.basePath, fixture.Name+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write fixture file: %w", err)
	}
	
	return nil
}

// ListFixtures returns all available fixtures
func (fm *fixtureManager) ListFixtures() ([]*TestFixture, error) {
	files, err := os.ReadDir(fm.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixtures directory: %w", err)
	}
	
	var fixtures []*TestFixture
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			name := file.Name()[:len(file.Name())-5] // Remove .json extension
			fixture, err := fm.LoadFixture(name)
			if err != nil {
				continue // Skip invalid fixtures
			}
			fixtures = append(fixtures, fixture)
		}
	}
	
	return fixtures, nil
}

// CreateTemporaryEnvironment sets up a temporary test environment
func (fm *fixtureManager) CreateTemporaryEnvironment(fixture *TestFixture) (string, error) {
	tmpDir, err := os.MkdirTemp("", "fixture_env_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	
	// Create files
	for filename, content := range fixture.Files {
		filePath := filepath.Join(tmpDir, filename)
		
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to create file %s: %w", filename, err)
		}
	}
	
	// Create config file if config exists
	if fixture.Config != nil {
		configPath := filepath.Join(tmpDir, "config.yaml")
		// Note: In a real implementation, we'd use yaml.Marshal
		configContent := "# Generated config file\n"
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to create config file: %w", err)
		}
	}
	
	return tmpDir, nil
}

// CleanupEnvironment cleans up a temporary test environment
func (fm *fixtureManager) CleanupEnvironment(path string) error {
	return os.RemoveAll(path)
}

// ValidateFixture validates a fixture's integrity
func (fm *fixtureManager) ValidateFixture(fixture *TestFixture) error {
	if fixture == nil {
		return fmt.Errorf("fixture cannot be nil")
	}
	
	if fixture.Name == "" {
		return fmt.Errorf("fixture name cannot be empty")
	}
	
	if fixture.Description == "" {
		return fmt.Errorf("fixture description cannot be empty")
	}
	
	if fixture.Config == nil {
		return fmt.Errorf("fixture config cannot be nil")
	}
	
	return nil
}

// GenerateFixtureFromProject generates a fixture from current project state
func (fm *fixtureManager) GenerateFixtureFromProject(name, description string) (*TestFixture, error) {
	fixture := &TestFixture{
		Name:        name,
		Description: description,
		Files:       make(map[string]string),
		Expected:    &ExpectedResults{},
		Metadata: &FixtureMetadata{
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Version:    "1.0.0",
			Tags:       []string{"generated"},
			Category:   "generated",
			Complexity: "simple",
			Author:     "fixture_generator",
		},
	}
	
	// Read files from the base path
	if err := filepath.Walk(fm.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".yaml")) {
			relPath, err := filepath.Rel(fm.basePath, path)
			if err != nil {
				return err
			}
			
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			fixture.Files[relPath] = string(content)
		}
		
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	// Create a basic config
	fixture.Config = &config.Config{
		Services: []config.ServiceRule{
			{
				ServiceName:   "generated_service",
				PackagePath:   "cloud.google.com/go/generated",
				CreationFuncs: []string{"NewClient"},
				CleanupMethods: []config.CleanupMethod{
					{
						Method:      "Close",
						Required:    true,
						Description: "Close the generated client",
					},
				},
			},
		},
	}
	
	return fixture, nil
}