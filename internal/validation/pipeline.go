package validation

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BuildResult represents the result of a build operation
type BuildResult struct {
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	BinaryPath string       `json:"binary_path,omitempty"`
}

// TestResult represents the result of test execution
type TestResult struct {
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Output      string        `json:"output"`
	Error       string        `json:"error,omitempty"`
	Passed      int          `json:"passed"`
	Failed      int          `json:"failed"`
	Skipped     int          `json:"skipped"`
	Coverage    float64      `json:"coverage,omitempty"`
	TestsPassed int          `json:"tests_passed"` // Alias for integration tests
	TestsFailed int          `json:"tests_failed"` // Alias for integration tests
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
	Success  bool                          `json:"success"`
	Duration time.Duration                 `json:"duration"`
	Build    BuildResult                   `json:"build"`
	Test     TestResult                    `json:"test"`
	Issues   []Issue                       `json:"issues,omitempty"`
	Output   string                        `json:"output"`
	Steps    map[string]QualityStepResult  `json:"steps"`
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
	Success       bool          `json:"success"`
	Summary       string        `json:"summary"`
	TotalDuration time.Duration `json:"total_duration"`
	Build         *BuildResult  `json:"build,omitempty"`
	Test          *TestResult   `json:"test,omitempty"`
	Quality       *QualityResult `json:"quality,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
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
	if err := exec.Command("mkdir", "-p", binDir).Run(); err != nil {
		return nil, fmt.Errorf("failed to create bin directory: %w", err)
	}
	
	// Parse build command
	cmdParts := strings.Fields(vp.buildCmd)
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
	
	// Parse test command
	cmdParts := strings.Fields(vp.testCmd)
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
	
	_, err := cmd.CombinedOutput()
	if err != nil {
		// golangci-lint returns non-zero exit code when issues found
		// We still want to parse the output
	}
	
	// For now, return empty slice - JSON parsing would be implemented here
	return []Issue{}, nil
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
	
	// Determine overall success
	if build != nil && !build.Success {
		report.Success = false
	}
	if test != nil && !test.Success {
		report.Success = false
	}
	if quality != nil && !quality.Success {
		report.Success = false
	}
	
	// Calculate total duration
	if build != nil {
		report.TotalDuration += build.Duration
	}
	if test != nil {
		report.TotalDuration += test.Duration
	}
	if quality != nil {
		report.TotalDuration += quality.Duration
	}
	
	// Generate summary
	if report.Success {
		report.Summary = "All validation checks passed successfully"
	} else {
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
		report.Summary = fmt.Sprintf("Validation failed: %s", strings.Join(failures, ", "))
	}
	
	return report
}