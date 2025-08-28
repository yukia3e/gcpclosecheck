package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildCLI builds the CLI binary for testing and returns the binary path and temp directory
func buildCLI(t *testing.T) (string, string) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gcpclosecheck")

	// Find the project root directory (where go.mod is located)
	// Start from current test's source file directory
	projectRoot := findProjectRoot(t)

	buildCmd := exec.Command("go", "build", "-o", binPath, ".") // #nosec G204 -- binPath is controlled temp directory for testing
	buildCmd.Dir = projectRoot                                   // Ensure build happens in project root
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, string(output))
	}
	return binPath, tmpDir
}

// findProjectRoot finds the project root directory containing go.mod
func findProjectRoot(t *testing.T) string {
	// Start from the cmd/gcpclosecheck directory and go up to find go.mod
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Look for go.mod in current dir and parent directories
	dir := currentDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, but we need the cmd/gcpclosecheck subdirectory
			return filepath.Join(dir, "cmd", "gcpclosecheck")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory without finding go.mod
			break
		}
		dir = parent
	}

	// Fallback: assume current directory is correct
	return currentDir
}
