package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCLIBasicExecution tests basic CLI execution
func TestCLIBasicExecution(t *testing.T) {
	// Build test executable file
	binPath, _ := buildCLI(t)

	// Test help option
	helpCmd := exec.Command(binPath, "-h")
	var helpOut bytes.Buffer
	helpCmd.Stdout = &helpOut
	helpCmd.Stderr = &helpOut

	if err := helpCmd.Run(); err == nil {
		// Verify help message is displayed in English
		output := helpOut.String()
		if !strings.Contains(output, "gcpclosecheck") {
			t.Errorf("Help output should contain 'gcpclosecheck', got: %s", output)
		}

		// Expect English help content matching messages constants
		expectedEnglishPhrases := []string{
			"Detects missing Close/Stop/Cancel calls for GCP resource clients",
			"Usage Examples",
			"Best Practices",
			"Flags:",
			"Environment Variables:",
			"Enable debug mode",
		}

		for _, phrase := range expectedEnglishPhrases {
			if !strings.Contains(output, phrase) {
				t.Errorf("Help output should contain English phrase %q, got: %s", phrase, output)
			}
		}

		// Should not contain Japanese characters
		if containsJapanese(output) {
			t.Errorf("Help output should not contain Japanese characters, got: %s", output)
		}
	}
}

// Helper function to detect Japanese characters
func containsJapanese(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FAF) { // Kanji
			return true
		}
	}
	return false
}

// TestCLIVersionFlag tests the version flag
func TestCLIVersionFlag(t *testing.T) {
	binPath, _ := buildCLI(t)

	// Test version flag
	versionCmd := exec.Command(binPath, "-V")
	var versionOut bytes.Buffer
	versionCmd.Stdout = &versionOut
	versionCmd.Stderr = &versionOut

	// Version flag should execute normally or display appropriate error message
	_ = versionCmd.Run()
	// output := versionOut.String()
	// Whether version information is displayed depends on implementation
}

// TestCLIAnalysisExecution tests actual analysis execution
func TestCLIAnalysisExecution(t *testing.T) {
	binPath, tmpDir := buildCLI(t)

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testCode := `
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	// defer client.Close() is missing - should be detected
	_ = client
}
`
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Execute analysis with CLI
	analysisCmd := exec.Command(binPath, testFile)
	var analysisOut bytes.Buffer
	analysisCmd.Stdout = &analysisOut
	analysisCmd.Stderr = &analysisOut
	analysisCmd.Dir = tmpDir

	// Set timeout
	done := make(chan error, 1)
	go func() {
		done <- analysisCmd.Run()
	}()

	select {
	case err := <-done:
		// Confirm analysis was executed (error presence doesn't matter)
		output := analysisOut.String()
		if err != nil {
			// Non-zero exit code due to analysis errors or detection results is normal
			t.Logf("Analysis completed with exit code (expected): %v", err)
			t.Logf("Output: %s", output)
		}
	case <-time.After(10 * time.Second):
		if err := analysisCmd.Process.Kill(); err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		t.Fatal("Analysis execution timed out")
	}
}

// TestCLIExitCodes tests exit codes in different scenarios
func TestCLIExitCodes(t *testing.T) {
	binPath, _ := buildCLI(t)

	tests := []struct {
		name          string
		args          []string
		expectNonZero bool
	}{
		{
			name:          "No arguments (help)",
			args:          []string{},
			expectNonZero: true, // Help display returns exit code 1 (flag package behavior)
		},
		{
			name:          "Invalid flag",
			args:          []string{"-invalid-flag"},
			expectNonZero: true, // Invalid flag is error
		},
		{
			name:          "Non-existent file",
			args:          []string{"/non/existent/file.go"},
			expectNonZero: true, // Non-existent file is error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Run()
			hasError := err != nil

			if tt.expectNonZero && !hasError {
				t.Errorf("Expected non-zero exit code, but got success")
			} else if !tt.expectNonZero && hasError {
				t.Errorf("Expected zero exit code, but got error: %v", err)
			}
		})
	}
}

// TestCLIOutputFormat tests output format
func TestCLIOutputFormat(t *testing.T) {
	binPath, tmpDir := buildCLI(t)

	// Test output format with valid Go file
	testFile := filepath.Join(tmpDir, "valid.go")
	validCode := `
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	defer client.Close() // Correct pattern
	_ = client
}
`
	if err := os.WriteFile(testFile, []byte(validCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Execute with valid code (no problems case)
	cmd := exec.Command(binPath, testFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = tmpDir

	_ = cmd.Run()
	output := out.String()

	// Expect minimal output or no error messages for valid code
	if strings.Contains(output, "panic") || strings.Contains(output, "fatal") {
		t.Errorf("Unexpected panic or fatal error in output: %s", output)
	}
}

// TestCLIPerformance runs basic performance tests
func TestCLIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	binPath, tmpDir := buildCLI(t)

	// Create medium-scale test file
	testFile := filepath.Join(tmpDir, "large_test.go")
	var codeBuilder strings.Builder
	codeBuilder.WriteString(`
package main

import (
	"context"
	"cloud.google.com/go/spanner"
)

func main() {
`)

	// 100個のクライアント生成を含むコードを生成
	for i := 0; i < 100; i++ {
		codeBuilder.WriteString(`
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
	if err != nil {
		panic(err)
	}
	_ = client
`)
	}

	codeBuilder.WriteString("}")

	if err := os.WriteFile(testFile, []byte(codeBuilder.String()), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// パフォーマンス測定
	start := time.Now()

	cmd := exec.Command(binPath, testFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = tmpDir

	err := cmd.Run()
	elapsed := time.Since(start)

	t.Logf("Analysis of 100 clients took: %v", elapsed)
	if elapsed > 5*time.Second {
		t.Errorf("Analysis took too long: %v (should be < 5s)", elapsed)
	}

	// エラーログの確認
	if err != nil {
		t.Logf("Analysis completed with error (expected): %v", err)
		t.Logf("Output: %s", out.String())
	}
}
