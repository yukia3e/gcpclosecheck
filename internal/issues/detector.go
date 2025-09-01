package issues

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
)

// Issue represents a quality issue found during linting
type Issue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Linter   string `json:"linter"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// IssueCategorization groups issues by different criteria
type IssueCategorization struct {
	ByLinter   map[string][]Issue `json:"by_linter"`
	BySeverity map[string][]Issue `json:"by_severity"`
	ByFile     map[string][]Issue `json:"by_file"`
	Total      int                `json:"total"`
}

// FixSuggestion represents a suggested fix for an issue
type FixSuggestion struct {
	Issue       Issue  `json:"issue"`
	Action      string `json:"action"`
	Message     string `json:"message"`
	AutoFixable bool   `json:"auto_fixable"`
}

// IssueCategory represents different types of linter categories
type IssueCategory string

const (
	CategoryErrorCheck    IssueCategory = "errcheck"
	CategorySecurity      IssueCategory = "gosec"
	CategoryComplexity    IssueCategory = "gocyclo"
	CategoryIneffAssign   IssueCategory = "ineffassign"
	CategoryUnused        IssueCategory = "unused"
	CategoryGoVet         IssueCategory = "govet"
	CategoryOther         IssueCategory = "other"
)

// severityPriority defines priority order for severity levels
var severityPriority = map[string]int{
	"error":   1,
	"warning": 2,
	"info":    3,
}

// linterPriority defines priority order for different linters
var linterPriority = map[string]int{
	"errcheck":    1, // Error handling is critical
	"gosec":       2, // Security issues are high priority
	"govet":       3, // Static analysis issues
	"gocyclo":     4, // Code complexity
	"ineffassign": 5, // Code quality
	"unused":      6, // Cleanup issues
}

// GolangciLintOutput represents the structure of golangci-lint JSON output
type GolangciLintOutput struct {
	Issues []GolangciLintIssue `json:"Issues"`
}

// GolangciLintIssue represents a single issue from golangci-lint
type GolangciLintIssue struct {
	FromLinter string                  `json:"FromLinter"`
	Text       string                  `json:"Text"`
	Severity   string                  `json:"Severity"`
	Pos        GolangciLintPosition    `json:"Pos"`
}

// GolangciLintPosition represents position information from golangci-lint
type GolangciLintPosition struct {
	Filename string `json:"Filename"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

// IssueDetector manages linter issue detection and parsing
type IssueDetector interface {
	// DetectIssues runs golangci-lint and returns parsed issues
	DetectIssues() ([]Issue, error)
	
	// ParseLinterOutput parses golangci-lint JSON output into Issue structs
	ParseLinterOutput(output string) ([]Issue, error)
	
	// CategorizeIssues groups issues by linter, severity, and file
	CategorizeIssues(issues []Issue) *IssueCategorization
	
	// PrioritizeIssues orders issues by severity and fixability
	PrioritizeIssues(issues []Issue) []Issue
	
	// GenerateFixSuggestions creates fix suggestions for common linter issues
	GenerateFixSuggestions(issues []Issue) ([]FixSuggestion, error)
	
	// ApplyAutoFix applies an automatic fix for fixable issues
	ApplyAutoFix(suggestion FixSuggestion) error
}

// issueDetector is the concrete implementation
type issueDetector struct {
	workDir string
}

// NewIssueDetector creates a new issue detector instance
func NewIssueDetector(workDir string) IssueDetector {
	return &issueDetector{
		workDir: workDir,
	}
}

// DetectIssues executes golangci-lint and returns parsed issues
func (id *issueDetector) DetectIssues() ([]Issue, error) {
	// Execute golangci-lint with JSON output
	cmd := exec.Command("golangci-lint", "run", "--out-format", "json")
	cmd.Dir = id.workDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// golangci-lint returns non-zero exit code when issues are found
		// We still want to parse the output if it contains valid JSON
		if len(output) == 0 {
			return nil, fmt.Errorf("golangci-lint execution failed: %w", err)
		}
	}
	
	return id.ParseLinterOutput(string(output))
}

// ParseLinterOutput parses golangci-lint JSON output into Issue structs
func (id *issueDetector) ParseLinterOutput(output string) ([]Issue, error) {
	if output == "" {
		return []Issue{}, nil
	}
	
	var lintOutput GolangciLintOutput
	if err := json.Unmarshal([]byte(output), &lintOutput); err != nil {
		return nil, fmt.Errorf("failed to parse golangci-lint output: %w", err)
	}
	
	issues := make([]Issue, len(lintOutput.Issues))
	for i, lintIssue := range lintOutput.Issues {
		issues[i] = Issue{
			File:     filepath.Base(lintIssue.Pos.Filename), // Use relative path for consistency
			Line:     lintIssue.Pos.Line,
			Column:   lintIssue.Pos.Column,
			Linter:   lintIssue.FromLinter,
			Message:  lintIssue.Text,
			Severity: lintIssue.Severity,
		}
	}
	
	return issues, nil
}

// CategorizeIssues groups issues by linter, severity, and file
func (id *issueDetector) CategorizeIssues(issues []Issue) *IssueCategorization {
	categorization := &IssueCategorization{
		ByLinter:   make(map[string][]Issue),
		BySeverity: make(map[string][]Issue),
		ByFile:     make(map[string][]Issue),
		Total:      len(issues),
	}
	
	for _, issue := range issues {
		// Categorize by linter
		categorization.ByLinter[issue.Linter] = append(categorization.ByLinter[issue.Linter], issue)
		
		// Categorize by severity
		categorization.BySeverity[issue.Severity] = append(categorization.BySeverity[issue.Severity], issue)
		
		// Categorize by file
		categorization.ByFile[issue.File] = append(categorization.ByFile[issue.File], issue)
	}
	
	return categorization
}

// PrioritizeIssues orders issues by severity and fixability
func (id *issueDetector) PrioritizeIssues(issues []Issue) []Issue {
	// Create a copy to avoid modifying the original slice
	prioritized := make([]Issue, len(issues))
	copy(prioritized, issues)
	
	// Sort by priority: severity first, then linter priority
	sort.Slice(prioritized, func(i, j int) bool {
		issueA, issueB := prioritized[i], prioritized[j]
		
		// Compare severity priority
		severityA := severityPriority[issueA.Severity]
		severityB := severityPriority[issueB.Severity]
		
		if severityA != severityB {
			return severityA < severityB // Lower number = higher priority
		}
		
		// If severity is the same, compare linter priority
		linterA := linterPriority[issueA.Linter]
		linterB := linterPriority[issueB.Linter]
		
		if linterA == 0 {
			linterA = 999 // Unknown linters get low priority
		}
		if linterB == 0 {
			linterB = 999
		}
		
		return linterA < linterB
	})
	
	return prioritized
}

// GenerateFixSuggestions creates fix suggestions for common linter issues
func (id *issueDetector) GenerateFixSuggestions(issues []Issue) ([]FixSuggestion, error) {
	suggestions := make([]FixSuggestion, 0, len(issues))
	
	for _, issue := range issues {
		suggestion := id.generateSuggestionForIssue(issue)
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

// generateSuggestionForIssue creates a fix suggestion for a specific issue
func (id *issueDetector) generateSuggestionForIssue(issue Issue) FixSuggestion {
	switch issue.Linter {
	case "errcheck":
		return FixSuggestion{
			Issue:       issue,
			Action:      "add_error_check",
			Message:     "Add proper error handling: if err != nil { ... }",
			AutoFixable: false, // Requires context-specific handling
		}
	
	case "goimports":
		return FixSuggestion{
			Issue:       issue,
			Action:      "run_goimports",
			Message:     "Run 'goimports -w' to fix import formatting",
			AutoFixable: true,
		}
	
	case "gofmt":
		return FixSuggestion{
			Issue:       issue,
			Action:      "run_gofmt",
			Message:     "Run 'gofmt -w' to fix code formatting",
			AutoFixable: true,
		}
	
	case "unused":
		return FixSuggestion{
			Issue:       issue,
			Action:      "remove_unused",
			Message:     "Remove unused variable/import/function",
			AutoFixable: false, // May affect other code
		}
	
	case "ineffassign":
		return FixSuggestion{
			Issue:       issue,
			Action:      "remove_assignment",
			Message:     "Remove ineffectual assignment",
			AutoFixable: false, // Requires code review
		}
	
	case "gosec":
		return FixSuggestion{
			Issue:       issue,
			Action:      "security_review",
			Message:     "Review security implications and add appropriate mitigations",
			AutoFixable: false, // Requires manual security analysis
		}
	
	default:
		return FixSuggestion{
			Issue:       issue,
			Action:      "manual_review",
			Message:     "Manual review required for this linter issue",
			AutoFixable: false,
		}
	}
}

// ApplyAutoFix applies an automatic fix for fixable issues
func (id *issueDetector) ApplyAutoFix(suggestion FixSuggestion) error {
	if !suggestion.AutoFixable {
		return fmt.Errorf("issue is not auto-fixable: %s", suggestion.Issue.Message)
	}
	
	switch suggestion.Action {
	case "run_goimports":
		return id.runGoImports(suggestion.Issue.File)
	
	case "run_gofmt":
		return id.runGoFmt(suggestion.Issue.File)
	
	default:
		return fmt.Errorf("unknown auto-fix action: %s", suggestion.Action)
	}
}

// runGoImports executes goimports on the specified file
func (id *issueDetector) runGoImports(file string) error {
	filePath := filepath.Join(id.workDir, file)
	cmd := exec.Command("goimports", "-w", filePath)
	cmd.Dir = id.workDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run goimports on %s: %w", file, err)
	}
	
	return nil
}

// runGoFmt executes gofmt on the specified file
func (id *issueDetector) runGoFmt(file string) error {
	filePath := filepath.Join(id.workDir, file)
	cmd := exec.Command("gofmt", "-w", filePath)
	cmd.Dir = id.workDir
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run gofmt on %s: %w", file, err)
	}
	
	return nil
}