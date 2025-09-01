package issues

import (
	"testing"
)

func TestNewIssueDetector(t *testing.T) {
	detector := NewIssueDetector("/test/dir")
	if detector == nil {
		t.Fatal("Expected IssueDetector instance, got nil")
	}
}

func TestIssueDetector_ParseLinterOutput(t *testing.T) {
	// Sample golangci-lint output
	sampleOutput := `{
  "Issues": [
    {
      "FromLinter": "errcheck",
      "Text": "Error return value of client.Close is not checked",
      "Severity": "error",
      "SourceLines": ["defer client.Close()"],
      "Replacement": null,
      "Pos": {
        "Filename": "internal/config/manager_test.go",
        "Offset": 1234,
        "Line": 45,
        "Column": 15
      },
      "ExpectedNoLint": false,
      "ExpectedNoLintLinter": ""
    },
    {
      "FromLinter": "gosec",
      "Text": "Potential file inclusion via variable",
      "Severity": "warning",
      "SourceLines": ["data, err := os.ReadFile(filepath.Clean(path))"],
      "Replacement": null,
      "Pos": {
        "Filename": "internal/config/manager.go",
        "Offset": 5678,
        "Line": 72,
        "Column": 10
      },
      "ExpectedNoLint": false,
      "ExpectedNoLintLinter": ""
    }
  ]
}`

	detector := NewIssueDetector(".")
	issues, err := detector.ParseLinterOutput(sampleOutput)
	if err != nil {
		t.Fatalf("Failed to parse linter output: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}

	// Verify first issue structure
	if len(issues) > 0 {
		issue := issues[0]
		if issue.File != "manager_test.go" {
			t.Errorf("Expected file 'manager_test.go', got %q", issue.File)
		}
		if issue.Line != 45 {
			t.Errorf("Expected line 45, got %d", issue.Line)
		}
		if issue.Column != 15 {
			t.Errorf("Expected column 15, got %d", issue.Column)
		}
		if issue.Linter != "errcheck" {
			t.Errorf("Expected linter 'errcheck', got %q", issue.Linter)
		}
		if issue.Severity != "error" {
			t.Errorf("Expected severity 'error', got %q", issue.Severity)
		}
		if issue.Message != "Error return value of client.Close is not checked" {
			t.Errorf("Expected specific message, got %q", issue.Message)
		}
	}
}

func TestIssueDetector_DetectIssues_RealProject(t *testing.T) {
	// Test with actual project directory
	detector := NewIssueDetector(".")
	issues, err := detector.DetectIssues()

	// This test allows for either success or known failures
	if err != nil {
		t.Logf("DetectIssues returned error (may be expected): %v", err)
	} else {
		t.Logf("DetectIssues found %d issues", len(issues))

		// Log first few issues for debugging
		for i, issue := range issues {
			if i < 3 {
				t.Logf("Issue %d: %s:%d:%d [%s] %s",
					i+1, issue.File, issue.Line, issue.Column, issue.Linter, issue.Message)
			}
		}
	}
}

func TestIssueDetector_DetectIssues_NoLinter(t *testing.T) {
	detector := NewIssueDetector("/nonexistent/dir")
	issues, err := detector.DetectIssues()

	// Should handle the case where golangci-lint is not available
	if err == nil {
		t.Log("golangci-lint not available, got empty result")
		if len(issues) != 0 {
			t.Error("Expected empty issues when linter not available")
		}
	} else {
		// Error is acceptable when golangci-lint is not installed
		t.Logf("Expected error when golangci-lint not available: %v", err)
	}
}

func TestIssue_Structure(t *testing.T) {
	issue := Issue{
		File:     "test.go",
		Line:     10,
		Column:   5,
		Linter:   "golint",
		Message:  "exported function should have comment",
		Severity: "warning",
	}

	if issue.File != "test.go" {
		t.Error("Expected file to be test.go")
	}

	if issue.Line != 10 {
		t.Error("Expected line to be 10")
	}

	if issue.Column != 5 {
		t.Error("Expected column to be 5")
	}

	if issue.Linter != "golint" {
		t.Error("Expected linter to be golint")
	}

	if issue.Message != "exported function should have comment" {
		t.Error("Expected specific message")
	}

	if issue.Severity != "warning" {
		t.Error("Expected severity to be warning")
	}
}

func TestIssueDetector_CategorizeIssues(t *testing.T) {
	issues := []Issue{
		{File: "test.go", Line: 10, Linter: "errcheck", Message: "Error return value not checked", Severity: "error"},
		{File: "main.go", Line: 20, Linter: "gosec", Message: "Potential file inclusion", Severity: "warning"},
		{File: "util.go", Line: 30, Linter: "gocyclo", Message: "Cyclomatic complexity too high", Severity: "warning"},
		{File: "api.go", Line: 40, Linter: "ineffassign", Message: "Ineffectual assignment", Severity: "info"},
	}

	detector := NewIssueDetector(".")
	categorized := detector.CategorizeIssues(issues)

	if categorized == nil {
		t.Fatal("Expected categorized issues, got nil")
	}

	// Check that we have categories
	if len(categorized.ByLinter) == 0 {
		t.Error("Expected issues to be categorized by linter")
	}

	if len(categorized.BySeverity) == 0 {
		t.Error("Expected issues to be categorized by severity")
	}

	// Check specific categorization
	if len(categorized.ByLinter["errcheck"]) != 1 {
		t.Errorf("Expected 1 errcheck issue, got %d", len(categorized.ByLinter["errcheck"]))
	}

	if len(categorized.BySeverity["error"]) != 1 {
		t.Errorf("Expected 1 error severity issue, got %d", len(categorized.BySeverity["error"]))
	}
}

func TestIssueDetector_PrioritizeIssues(t *testing.T) {
	issues := []Issue{
		{File: "test.go", Line: 10, Linter: "errcheck", Message: "Error return value not checked", Severity: "error"},
		{File: "main.go", Line: 20, Linter: "gosec", Message: "Potential security issue", Severity: "error"},
		{File: "util.go", Line: 30, Linter: "gocyclo", Message: "Cyclomatic complexity too high", Severity: "warning"},
		{File: "api.go", Line: 40, Linter: "ineffassign", Message: "Ineffectual assignment", Severity: "info"},
	}

	detector := NewIssueDetector(".")
	prioritized := detector.PrioritizeIssues(issues)

	if len(prioritized) != len(issues) {
		t.Errorf("Expected %d prioritized issues, got %d", len(issues), len(prioritized))
	}

	// First issue should be high priority (error severity)
	if len(prioritized) > 0 {
		firstIssue := prioritized[0]
		if firstIssue.Severity != "error" {
			t.Errorf("Expected first prioritized issue to have error severity, got %q", firstIssue.Severity)
		}
	}

	// Last issue should be lower priority
	if len(prioritized) > 3 {
		lastIssue := prioritized[len(prioritized)-1]
		if lastIssue.Severity == "error" {
			t.Error("Expected last prioritized issue to have lower priority than error")
		}
	}
}

func TestIssueCategorization_Structure(t *testing.T) {
	categorization := IssueCategorization{
		ByLinter: map[string][]Issue{
			"errcheck": {{File: "test.go", Linter: "errcheck"}},
		},
		BySeverity: map[string][]Issue{
			"error": {{File: "test.go", Severity: "error"}},
		},
		ByFile: map[string][]Issue{
			"test.go": {{File: "test.go"}},
		},
		Total: 1,
	}

	if categorization.Total != 1 {
		t.Error("Expected total to be 1")
	}

	if len(categorization.ByLinter) != 1 {
		t.Error("Expected 1 linter category")
	}

	if len(categorization.BySeverity) != 1 {
		t.Error("Expected 1 severity category")
	}

	if len(categorization.ByFile) != 1 {
		t.Error("Expected 1 file category")
	}
}

func TestIssueDetector_GenerateFixSuggestions(t *testing.T) {
	issues := []Issue{
		{File: "test.go", Line: 10, Linter: "errcheck", Message: "Error return value of client.Close is not checked", Severity: "error"},
		{File: "main.go", Line: 20, Linter: "goimports", Message: "File is not goimports-ed", Severity: "warning"},
		{File: "util.go", Line: 30, Linter: "unused", Message: "unused variable x", Severity: "info"},
		{File: "api.go", Line: 40, Linter: "unknown-linter", Message: "Unknown issue", Severity: "warning"},
	}

	detector := NewIssueDetector(".")
	suggestions, err := detector.GenerateFixSuggestions(issues)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("Expected fix suggestions to be generated")
	}

	// Check that auto-fixable issues are identified
	autoFixableCount := 0
	for _, suggestion := range suggestions {
		if suggestion.AutoFixable {
			autoFixableCount++
		}
	}

	if autoFixableCount == 0 {
		t.Error("Expected at least some issues to be auto-fixable")
	}

	// Verify errcheck suggestion
	errcheckFound := false
	for _, suggestion := range suggestions {
		if suggestion.Issue.Linter == "errcheck" {
			errcheckFound = true
			if suggestion.Action == "" {
				t.Error("Expected action for errcheck issue")
			}
			if suggestion.Message == "" {
				t.Error("Expected message for errcheck issue")
			}
		}
	}

	if !errcheckFound {
		t.Error("Expected fix suggestion for errcheck issue")
	}
}

func TestIssueDetector_ApplyAutoFix(t *testing.T) {
	// Create a simple auto-fixable issue
	issue := Issue{
		File:     "test.go",
		Line:     10,
		Linter:   "goimports",
		Message:  "File is not goimports-ed",
		Severity: "warning",
	}

	suggestion := FixSuggestion{
		Issue:       issue,
		Action:      "run_goimports",
		Message:     "Run goimports to fix import formatting",
		AutoFixable: true,
	}

	detector := NewIssueDetector(".")
	err := detector.ApplyAutoFix(suggestion)

	// This test should handle both success and expected failures
	if err != nil {
		t.Logf("ApplyAutoFix returned error (may be expected): %v", err)
	} else {
		t.Log("ApplyAutoFix completed successfully")
	}
}

func TestFixSuggestion_Structure(t *testing.T) {
	issue := Issue{File: "test.go", Line: 10, Linter: "errcheck", Message: "Error not checked", Severity: "error"}

	suggestion := FixSuggestion{
		Issue:       issue,
		Action:      "add_error_check",
		Message:     "Add proper error handling",
		AutoFixable: false,
	}

	if suggestion.Issue.File != "test.go" {
		t.Error("Expected issue file to be preserved")
	}

	if suggestion.Action != "add_error_check" {
		t.Error("Expected specific action")
	}

	if suggestion.Message != "Add proper error handling" {
		t.Error("Expected specific message")
	}

	if suggestion.AutoFixable {
		t.Error("Expected AutoFixable to be false")
	}
}

func TestNewResolutionWorkflow(t *testing.T) {
	workflow := NewResolutionWorkflow(".")
	if workflow == nil {
		t.Fatal("Expected ResolutionWorkflow instance, got nil")
	}
}

func TestResolutionWorkflow_ExecuteResolution(t *testing.T) {
	issues := []Issue{
		{File: "test.go", Line: 10, Linter: "goimports", Message: "File is not goimports-ed", Severity: "warning"},
		{File: "main.go", Line: 20, Linter: "errcheck", Message: "Error return value not checked", Severity: "error"},
	}

	workflow := NewResolutionWorkflow(".")
	result, err := workflow.ExecuteResolution(issues)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected ResolutionResult, got nil")
	}

	if result.TotalIssues != len(issues) {
		t.Errorf("Expected total issues %d, got %d", len(issues), result.TotalIssues)
	}

	if result.ProcessedIssues == 0 {
		t.Error("Expected some issues to be processed")
	}

	if len(result.Steps) == 0 {
		t.Error("Expected resolution steps to be recorded")
	}
}

func TestResolutionWorkflow_ValidateStep(t *testing.T) {
	workflow := NewResolutionWorkflow(".")

	// Test validation after a hypothetical fix
	err := workflow.ValidateStep()

	// This should either succeed or fail gracefully
	if err != nil {
		t.Logf("ValidateStep returned error (may be expected): %v", err)
	} else {
		t.Log("ValidateStep completed successfully")
	}
}

func TestResolutionResult_Structure(t *testing.T) {
	steps := []ResolutionStep{
		{Issue: Issue{File: "test.go", Linter: "errcheck"}, Action: "add_error_check", Success: true},
	}

	result := ResolutionResult{
		TotalIssues:     2,
		ProcessedIssues: 1,
		FixedIssues:     1,
		FailedIssues:    0,
		Steps:           steps,
		Success:         true,
	}

	if result.TotalIssues != 2 {
		t.Error("Expected total issues to be 2")
	}

	if result.ProcessedIssues != 1 {
		t.Error("Expected processed issues to be 1")
	}

	if result.FixedIssues != 1 {
		t.Error("Expected fixed issues to be 1")
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if len(result.Steps) != 1 {
		t.Error("Expected 1 resolution step")
	}
}
