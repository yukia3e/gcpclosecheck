package ci

import (
	"testing"
	"time"
)

func TestNewCIFailureAnalyzer(t *testing.T) {
	analyzer := NewCIFailureAnalyzer("test-repo")
	if analyzer == nil {
		t.Fatal("Expected CIFailureAnalyzer instance, got nil")
	}
}

func TestCIFailureAnalyzer_AnalyzeFailure(t *testing.T) {
	// Sample GitHub Actions log with failure
	sampleLog := `2023-01-01T12:00:00.0000000Z ##[group]Run go test ./...
2023-01-01T12:00:01.0000000Z go test ./...
2023-01-01T12:00:05.0000000Z --- FAIL: TestSomething (0.00s)
2023-01-01T12:00:05.0000000Z     test_test.go:10: Expected value to be 5, got 3
2023-01-01T12:00:05.0000000Z FAIL	github.com/test/package	0.123s
2023-01-01T12:00:05.0000000Z ##[error]Process completed with exit code 1.`
	
	analyzer := NewCIFailureAnalyzer("test-repo")
	analysis, err := analyzer.AnalyzeFailure("test-job", sampleLog)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if analysis == nil {
		t.Fatal("Expected FailureAnalysis, got nil")
	}
	
	if analysis.JobName != "test-job" {
		t.Errorf("Expected JobName 'test-job', got %s", analysis.JobName)
	}
	
	if analysis.FailureType == "" {
		t.Error("Expected FailureType to be identified")
	}
	
	if len(analysis.ErrorLines) == 0 {
		t.Error("Expected error lines to be extracted")
	}
	
	if len(analysis.Suggestions) == 0 {
		t.Error("Expected suggestions to be generated")
	}
}

func TestCIFailureAnalyzer_IdentifyFailureType(t *testing.T) {
	analyzer := NewCIFailureAnalyzer("test-repo")
	
	testCases := []struct {
		name         string
		logContent   string
		expectedType FailureType
	}{
		{
			name:         "Test failure",
			logContent:   "FAIL TestSomething (0.00s)",
			expectedType: TestFailure,
		},
		{
			name:         "Build failure",
			logContent:   "build failed: syntax error",
			expectedType: BuildFailure,
		},
		{
			name:         "Lint failure",
			logContent:   "golangci-lint run failed",
			expectedType: LintFailure,
		},
		{
			name:         "Timeout failure",
			logContent:   "The job running on runner GitHub Actions has exceeded the maximum execution time",
			expectedType: TimeoutFailure,
		},
		{
			name:         "Unknown failure",
			logContent:   "Some unknown error occurred",
			expectedType: UnknownFailure,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			failureType := analyzer.IdentifyFailureType(tc.logContent)
			if failureType != tc.expectedType {
				t.Errorf("Expected failure type %v, got %v", tc.expectedType, failureType)
			}
		})
	}
}

func TestCIFailureAnalyzer_ExtractErrorLines(t *testing.T) {
	sampleLog := `2023-01-01T12:00:00.0000000Z ##[group]Run tests
2023-01-01T12:00:01.0000000Z Normal log line
2023-01-01T12:00:02.0000000Z --- FAIL: TestSomething (0.00s)
2023-01-01T12:00:03.0000000Z     test.go:10: assertion failed
2023-01-01T12:00:04.0000000Z ##[error]Test failed
2023-01-01T12:00:05.0000000Z Another normal line
2023-01-01T12:00:06.0000000Z panic: runtime error`
	
	analyzer := NewCIFailureAnalyzer("test-repo")
	errorLines := analyzer.ExtractErrorLines(sampleLog)
	
	if len(errorLines) == 0 {
		t.Error("Expected error lines to be extracted")
	}
	
	// Should contain the FAIL line
	failFound := false
	errorFound := false
	panicFound := false
	
	for _, line := range errorLines {
		if line.Contains("FAIL: TestSomething") {
			failFound = true
		}
		if line.Contains("##[error]Test failed") {
			errorFound = true
		}
		if line.Contains("panic: runtime error") {
			panicFound = true
		}
	}
	
	if !failFound {
		t.Error("Expected FAIL line to be identified")
	}
	if !errorFound {
		t.Error("Expected ##[error] line to be identified")
	}
	if !panicFound {
		t.Error("Expected panic line to be identified")
	}
}

func TestCIFailureAnalyzer_GenerateSuggestions(t *testing.T) {
	analyzer := NewCIFailureAnalyzer("test-repo")
	
	testFailureAnalysis := FailureAnalysis{
		JobName:     "test",
		FailureType: TestFailure,
		ErrorLines:  []ErrorLine{{Content: "TestSomething failed", LineNumber: 10}},
		Summary:     "Test failure detected",
	}
	
	suggestions := analyzer.GenerateSuggestions(testFailureAnalysis)
	
	if len(suggestions) == 0 {
		t.Error("Expected suggestions to be generated for test failure")
	}
	
	buildFailureAnalysis := FailureAnalysis{
		JobName:     "build",
		FailureType: BuildFailure,
		ErrorLines:  []ErrorLine{{Content: "syntax error", LineNumber: 5}},
		Summary:     "Build failure detected",
	}
	
	buildSuggestions := analyzer.GenerateSuggestions(buildFailureAnalysis)
	
	if len(buildSuggestions) == 0 {
		t.Error("Expected suggestions to be generated for build failure")
	}
}

func TestCIFailureAnalyzer_RecordFailurePattern(t *testing.T) {
	analyzer := NewCIFailureAnalyzer("test-repo")
	
	pattern := FailurePattern{
		Type:        TestFailure,
		Pattern:     "FAIL TestSomething",
		Frequency:   1,
		LastSeen:    time.Now(),
		Description: "Test failure in TestSomething",
	}
	
	err := analyzer.RecordFailurePattern(pattern)
	if err != nil {
		t.Fatalf("Unexpected error recording failure pattern: %v", err)
	}
	
	// Record the same pattern again to test frequency increment
	err = analyzer.RecordFailurePattern(pattern)
	if err != nil {
		t.Fatalf("Unexpected error recording duplicate failure pattern: %v", err)
	}
	
	// Get recorded patterns
	patterns := analyzer.GetFailurePatterns()
	if len(patterns) == 0 {
		t.Error("Expected recorded failure patterns")
	}
	
	// Find our pattern and verify frequency
	found := false
	for _, p := range patterns {
		if p.Pattern == "FAIL TestSomething" {
			found = true
			if p.Frequency < 2 {
				t.Errorf("Expected frequency >= 2, got %d", p.Frequency)
			}
		}
	}
	
	if !found {
		t.Error("Expected to find recorded failure pattern")
	}
}

func TestFailureAnalysis_Structure(t *testing.T) {
	errorLines := []ErrorLine{
		{Content: "Error line 1", LineNumber: 10, Severity: "error"},
		{Content: "Warning line 2", LineNumber: 15, Severity: "warning"},
	}
	
	analysis := FailureAnalysis{
		JobName:     "test-job",
		FailureType: TestFailure,
		ErrorLines:  errorLines,
		Summary:     "Test failure analysis",
		Suggestions: []string{"Fix test", "Check logic"},
		Timestamp:   time.Now(),
	}
	
	if analysis.JobName != "test-job" {
		t.Error("Expected JobName to be test-job")
	}
	
	if analysis.FailureType != TestFailure {
		t.Error("Expected FailureType to be TestFailure")
	}
	
	if len(analysis.ErrorLines) != 2 {
		t.Error("Expected 2 error lines")
	}
	
	if len(analysis.Suggestions) != 2 {
		t.Error("Expected 2 suggestions")
	}
}

func TestErrorLine_Structure(t *testing.T) {
	errorLine := ErrorLine{
		Content:    "test failed: assertion error",
		LineNumber: 42,
		Severity:   "error",
		Timestamp:  time.Now(),
	}
	
	if errorLine.Content != "test failed: assertion error" {
		t.Error("Expected specific content")
	}
	
	if errorLine.LineNumber != 42 {
		t.Error("Expected LineNumber to be 42")
	}
	
	if errorLine.Severity != "error" {
		t.Error("Expected Severity to be error")
	}
}

func TestErrorLine_Contains(t *testing.T) {
	errorLine := ErrorLine{
		Content: "FAIL TestSomething (0.00s)",
	}
	
	if !errorLine.Contains("FAIL") {
		t.Error("Expected Contains('FAIL') to return true")
	}
	
	if !errorLine.Contains("TestSomething") {
		t.Error("Expected Contains('TestSomething') to return true")
	}
	
	if errorLine.Contains("NotFound") {
		t.Error("Expected Contains('NotFound') to return false")
	}
}

func TestFailurePattern_Structure(t *testing.T) {
	pattern := FailurePattern{
		Type:        BuildFailure,
		Pattern:     "build failed: syntax error",
		Frequency:   5,
		LastSeen:    time.Now(),
		Description: "Common syntax error in build",
	}
	
	if pattern.Type != BuildFailure {
		t.Error("Expected Type to be BuildFailure")
	}
	
	if pattern.Frequency != 5 {
		t.Error("Expected Frequency to be 5")
	}
	
	if pattern.Description == "" {
		t.Error("Expected Description to be set")
	}
}