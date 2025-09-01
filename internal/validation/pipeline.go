package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BuildResult represents the result of a build operation
type BuildResult struct {
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	Output     string        `json:"output"`
	Error      string        `json:"error,omitempty"`
	BinaryPath string        `json:"binary_path,omitempty"`
}

// TestResult represents the result of test execution
type TestResult struct {
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Output      string        `json:"output"`
	Error       string        `json:"error,omitempty"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	Coverage    float64       `json:"coverage,omitempty"`
	TestsPassed int           `json:"tests_passed"` // Alias for integration tests
	TestsFailed int           `json:"tests_failed"` // Alias for integration tests
}

// QualityStepResult represents the result of a single quality check step
type QualityStepResult struct {
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
}

// QualityResult aggregates quality check results
type QualityResult struct {
	Success  bool                         `json:"success"`
	Duration time.Duration                `json:"duration"`
	Build    BuildResult                  `json:"build"`
	Test     TestResult                   `json:"test"`
	Issues   []Issue                      `json:"issues,omitempty"`
	Output   string                       `json:"output"`
	Steps    map[string]QualityStepResult `json:"steps"`
}

// Issue represents a quality issue found during checks
type Issue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Linter   string `json:"linter"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ValidationReport represents the overall result of validation
type ValidationReport struct {
	Success       bool           `json:"success"`
	Summary       string         `json:"summary"`
	TotalDuration time.Duration  `json:"total_duration"`
	Build         *BuildResult   `json:"build,omitempty"`
	Test          *TestResult    `json:"test,omitempty"`
	Quality       *QualityResult `json:"quality,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

// ValidationPipeline manages build, test, and quality validation workflows
type ValidationPipeline interface {
	// RunBuild executes go build command and returns result
	RunBuild() (*BuildResult, error)

	// RunTests executes go test command and parses results
	RunTests() (*TestResult, error)

	// RunQualityChecks executes comprehensive quality checks
	RunQualityChecks() (*QualityResult, error)

	// RunQualityChecksWithTargets runs quality checks with specific make targets
	RunQualityChecksWithTargets(targets []string) (*QualityResult, error)

	// GenerateReport aggregates results into a comprehensive report
	GenerateReport(build *BuildResult, test *TestResult, quality *QualityResult) *ValidationReport
}

// CommandExecutor interface for abstracting command execution
type CommandExecutor interface {
	ExecuteCommand(name string, args ...string) ([]byte, error)
}

// defaultCommandExecutor is the standard implementation
type defaultCommandExecutor struct{}

func (d *defaultCommandExecutor) ExecuteCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// validationPipeline is the concrete implementation
type validationPipeline struct {
	workDir    string
	binaryName string
	buildCmd   string
	testCmd    string
	executor   CommandExecutor
}

// NewValidationPipeline creates a new validation pipeline instance
func NewValidationPipeline(workDir string) ValidationPipeline {
	return &validationPipeline{
		workDir:    workDir,
		binaryName: "gcpclosecheck",
		buildCmd:   "go build -o bin/gcpclosecheck ./cmd/gcpclosecheck",
		testCmd:    "go test -v ./...",
		executor:   &defaultCommandExecutor{},
	}
}

// NewValidationPipelineWithExecutor creates a new validation pipeline with custom executor
func NewValidationPipelineWithExecutor(workDir string, executor CommandExecutor) ValidationPipeline {
	return &validationPipeline{
		workDir:    workDir,
		binaryName: "gcpclosecheck",
		buildCmd:   "go build",
		testCmd:    "go test",
		executor:   executor,
	}
}

// RunBuild executes the build command and returns structured result
func (vp *validationPipeline) RunBuild() (*BuildResult, error) {
	start := time.Now()

	// Create bin directory if it doesn't exist
	binDir := filepath.Join(vp.workDir, "bin")
	if err := os.MkdirAll(binDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Parse build command with strict validation
	cmdParts := strings.Fields(vp.buildCmd)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty build command")
	}
	// Only allow safe commands and validate arguments
	if cmdParts[0] != "go" {
		return nil, fmt.Errorf("only go commands are allowed for build")
	}
	// Validate all command parts contain no dangerous characters
	for _, part := range cmdParts {
		if strings.ContainsAny(part, ";|&$`(){}[]<>*?~") {
			return nil, fmt.Errorf("unsafe characters in build command: %s", part)
		}
	}
	// #nosec G204: cmdParts are validated and safe
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = vp.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &BuildResult{
		Success:  err == nil,
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Error = err.Error()
	} else {
		result.BinaryPath = filepath.Join(binDir, vp.binaryName)
	}

	return result, nil
}

// RunTests executes test command and parses results
func (vp *validationPipeline) RunTests() (*TestResult, error) {
	start := time.Now()

	// Parse test command with strict validation
	cmdParts := strings.Fields(vp.testCmd)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty test command")
	}
	// Only allow safe commands and validate arguments
	if cmdParts[0] != "go" {
		return nil, fmt.Errorf("only go commands are allowed for tests")
	}
	// Validate all command parts contain no dangerous characters
	for _, part := range cmdParts {
		if strings.ContainsAny(part, ";|&$`(){}[]<>*?~") {
			return nil, fmt.Errorf("unsafe characters in test command: %s", part)
		}
	}
	// #nosec G204: cmdParts are validated and safe
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = vp.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &TestResult{
		Success:  err == nil,
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Parse test output for statistics
	vp.parseTestOutput(string(output), result)

	return result, nil
}

// RunQualityChecks executes comprehensive quality validation
func (vp *validationPipeline) RunQualityChecks() (*QualityResult, error) {
	start := time.Now()

	result := &QualityResult{
		Success: true,
	}

	// Run build
	buildResult, err := vp.RunBuild()
	if err != nil {
		return nil, fmt.Errorf("build execution failed: %w", err)
	}
	result.Build = *buildResult

	if !buildResult.Success {
		result.Success = false
	}

	// Run tests
	testResult, err := vp.RunTests()
	if err != nil {
		return nil, fmt.Errorf("test execution failed: %w", err)
	}
	result.Test = *testResult

	if !testResult.Success {
		result.Success = false
	}

	// Run linter (if available)
	issues, err := vp.runLinter()
	if err == nil {
		result.Issues = issues
		if len(issues) > 0 {
			result.Success = false
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// parseTestOutput extracts test statistics from go test output
func (vp *validationPipeline) parseTestOutput(output string, result *TestResult) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count individual test results from --- PASS/FAIL/SKIP lines
		if strings.HasPrefix(line, "--- PASS:") {
			result.Passed++
		}
		if strings.HasPrefix(line, "--- FAIL:") {
			result.Failed++
		}
		if strings.HasPrefix(line, "--- SKIP:") {
			result.Skipped++
		}

		// Look for coverage information
		if strings.Contains(line, "coverage:") {
			// Parse coverage percentage if present
			// Format: "coverage: XX.X% of statements"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					if coverage := parseFloat(coverageStr); coverage >= 0 {
						result.Coverage = coverage
					}
				}
			}
		}
	}
}

// runLinter executes golangci-lint and parses issues
func (vp *validationPipeline) runLinter() ([]Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--out-format", "json")
	cmd.Dir = vp.workDir

	output, err := cmd.CombinedOutput()
	// golangci-lint returns non-zero exit code when issues found
	// We still want to parse the output even if err != nil

	if len(output) == 0 {
		// No output means no issues found
		return []Issue{}, nil
	}

	// Parse golangci-lint JSON output
	var lintResult struct {
		Issues []struct {
			FromLinter  string      `json:"FromLinter"`
			Text        string      `json:"Text"`
			Severity    string      `json:"Severity"`
			SourceLines []string    `json:"SourceLines"`
			Replacement interface{} `json:"Replacement"`
			Pos         struct {
				Filename string `json:"Filename"`
				Offset   int    `json:"Offset"`
				Line     int    `json:"Line"`
				Column   int    `json:"Column"`
			} `json:"Pos"`
			ExpectNoLint         bool   `json:"ExpectNoLint"`
			ExpectedNoLintLinter string `json:"ExpectedNoLintLinter"`
		} `json:"Issues"`
	}

	if jsonErr := json.Unmarshal(output, &lintResult); jsonErr != nil {
		// If JSON parsing fails, still return the original error if there was one
		if err != nil {
			return nil, fmt.Errorf("golangci-lint execution failed: %w", err)
		}
		return nil, fmt.Errorf("failed to parse golangci-lint JSON output: %w", jsonErr)
	}

	// Convert to our Issue format
	var issues []Issue
	for _, lintIssue := range lintResult.Issues {
		issue := Issue{
			File:     lintIssue.Pos.Filename,
			Line:     lintIssue.Pos.Line,
			Column:   lintIssue.Pos.Column,
			Linter:   lintIssue.FromLinter,
			Message:  lintIssue.Text,
			Severity: lintIssue.Severity,
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// parseFloat safely parses a float string
func parseFloat(s string) float64 {
	var result float64
	n, err := fmt.Sscanf(s, "%f", &result)
	if n != 1 || err != nil {
		return -1
	}
	return result
}

// RunQualityChecksWithTargets runs quality checks with specific make targets
func (vp *validationPipeline) RunQualityChecksWithTargets(targets []string) (*QualityResult, error) {
	start := time.Now()

	result := &QualityResult{
		Success: true,
		Steps:   make(map[string]QualityStepResult),
	}

	// Run each target
	for _, target := range targets {
		stepStart := time.Now()

		output, err := vp.executor.ExecuteCommand("make", target)

		stepResult := QualityStepResult{
			Success:  err == nil,
			Duration: time.Since(stepStart),
			Output:   string(output),
		}

		if err != nil {
			stepResult.Error = err.Error()
			result.Success = false
		}

		result.Steps[target] = stepResult

		// Also update the overall output
		if result.Output != "" {
			result.Output += "\n"
		}
		result.Output += string(output)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// GenerateReport aggregates results into a comprehensive report
func (vp *validationPipeline) GenerateReport(build *BuildResult, test *TestResult, quality *QualityResult) *ValidationReport {
	report := &ValidationReport{
		Success:   true,
		Build:     build,
		Test:      test,
		Quality:   quality,
		Timestamp: time.Now(),
	}

	report.Success = vp.determineOverallSuccess(build, test, quality)
	report.TotalDuration = vp.calculateTotalDuration(build, test, quality)
	report.Summary = vp.generateSummary(report.Success, build, test, quality)

	return report
}

// determineOverallSuccess checks if all components succeeded
func (vp *validationPipeline) determineOverallSuccess(build *BuildResult, test *TestResult, quality *QualityResult) bool {
	if build != nil && !build.Success {
		return false
	}
	if test != nil && !test.Success {
		return false
	}
	if quality != nil && !quality.Success {
		return false
	}
	return true
}

// calculateTotalDuration sums up all component durations
func (vp *validationPipeline) calculateTotalDuration(build *BuildResult, test *TestResult, quality *QualityResult) time.Duration {
	var total time.Duration
	if build != nil {
		total += build.Duration
	}
	if test != nil {
		total += test.Duration
	}
	if quality != nil {
		total += quality.Duration
	}
	return total
}

// generateSummary creates a summary message based on success status and failures
func (vp *validationPipeline) generateSummary(success bool, build *BuildResult, test *TestResult, quality *QualityResult) string {
	if success {
		return "All validation checks passed successfully"
	}

	var failures []string
	if build != nil && !build.Success {
		failures = append(failures, "build")
	}
	if test != nil && !test.Success {
		failures = append(failures, "tests")
	}
	if quality != nil && !quality.Success {
		failures = append(failures, "quality checks")
	}
	return fmt.Sprintf("Validation failed: %s", strings.Join(failures, ", "))
}
