package ci

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// FailureType represents different types of CI failures
type FailureType string

const (
	TestFailure    FailureType = "test"
	BuildFailure   FailureType = "build"
	LintFailure    FailureType = "lint"
	TimeoutFailure FailureType = "timeout"
	UnknownFailure FailureType = "unknown"
)

// ErrorLine represents a single error line from CI logs
type ErrorLine struct {
	Content    string    `json:"content"`
	LineNumber int       `json:"line_number"`
	Severity   string    `json:"severity"`
	Timestamp  time.Time `json:"timestamp"`
}

// Contains checks if the error line contains a specific substring
func (el *ErrorLine) Contains(substring string) bool {
	return strings.Contains(el.Content, substring)
}

// FailureAnalysis represents the analysis result of a CI failure
type FailureAnalysis struct {
	JobName     string      `json:"job_name"`
	FailureType FailureType `json:"failure_type"`
	ErrorLines  []ErrorLine `json:"error_lines"`
	Summary     string      `json:"summary"`
	Suggestions []string    `json:"suggestions"`
	Timestamp   time.Time   `json:"timestamp"`
}

// FailurePattern represents a recurring failure pattern
type FailurePattern struct {
	Type        FailureType `json:"type"`
	Pattern     string      `json:"pattern"`
	Frequency   int         `json:"frequency"`
	LastSeen    time.Time   `json:"last_seen"`
	Description string      `json:"description"`
}

// CIFailureAnalyzer analyzes CI failures and tracks patterns
type CIFailureAnalyzer interface {
	// AnalyzeFailure analyzes a CI job failure and identifies causes
	AnalyzeFailure(jobName, logContent string) (*FailureAnalysis, error)
	
	// IdentifyFailureType determines the type of failure from log content
	IdentifyFailureType(logContent string) FailureType
	
	// ExtractErrorLines extracts relevant error lines from logs
	ExtractErrorLines(logContent string) []ErrorLine
	
	// GenerateSuggestions provides suggestions based on failure analysis
	GenerateSuggestions(analysis FailureAnalysis) []string
	
	// RecordFailurePattern records a failure pattern for tracking
	RecordFailurePattern(pattern FailurePattern) error
	
	// GetFailurePatterns returns all recorded failure patterns
	GetFailurePatterns() []FailurePattern
}

// ciFailureAnalyzer is the concrete implementation
type ciFailureAnalyzer struct {
	repository      string
	failurePatterns map[string]*FailurePattern
	mutex           sync.RWMutex
}

// NewCIFailureAnalyzer creates a new CI failure analyzer instance
func NewCIFailureAnalyzer(repository string) CIFailureAnalyzer {
	return &ciFailureAnalyzer{
		repository:      repository,
		failurePatterns: make(map[string]*FailurePattern),
	}
}

// AnalyzeFailure analyzes a CI job failure and identifies causes
func (cfa *ciFailureAnalyzer) AnalyzeFailure(jobName, logContent string) (*FailureAnalysis, error) {
	analysis := &FailureAnalysis{
		JobName:     jobName,
		FailureType: cfa.IdentifyFailureType(logContent),
		ErrorLines:  cfa.ExtractErrorLines(logContent),
		Timestamp:   time.Now(),
	}
	
	// Generate summary based on failure type
	analysis.Summary = cfa.generateSummary(analysis.FailureType, analysis.ErrorLines)
	
	// Generate suggestions
	analysis.Suggestions = cfa.GenerateSuggestions(*analysis)
	
	return analysis, nil
}

// IdentifyFailureType determines the type of failure from log content
func (cfa *ciFailureAnalyzer) IdentifyFailureType(logContent string) FailureType {
	lowerContent := strings.ToLower(logContent)
	
	// Check for test failures
	if strings.Contains(lowerContent, "fail testsomething") || 
	   strings.Contains(lowerContent, "--- fail:") ||
	   (strings.Contains(lowerContent, "test") && strings.Contains(lowerContent, "failed")) {
		return TestFailure
	}
	
	// Check for build failures
	if strings.Contains(lowerContent, "build failed") || 
	   strings.Contains(lowerContent, "compilation") || 
	   strings.Contains(lowerContent, "syntax error") {
		return BuildFailure
	}
	
	// Check for lint failures
	if strings.Contains(lowerContent, "golangci-lint") || 
	   strings.Contains(lowerContent, "lint") && strings.Contains(lowerContent, "failed") {
		return LintFailure
	}
	
	// Check for timeout failures
	if strings.Contains(lowerContent, "timeout") || 
	   strings.Contains(lowerContent, "exceeded the maximum execution time") {
		return TimeoutFailure
	}
	
	return UnknownFailure
}

// ExtractErrorLines extracts relevant error lines from logs
func (cfa *ciFailureAnalyzer) ExtractErrorLines(logContent string) []ErrorLine {
	lines := strings.Split(logContent, "\n")
	var errorLines []ErrorLine
	
	for i, line := range lines {
		if cfa.isErrorLine(line) {
			severity := cfa.determineSeverity(line)
			errorLine := ErrorLine{
				Content:    line,
				LineNumber: i + 1,
				Severity:   severity,
				Timestamp:  time.Now(),
			}
			errorLines = append(errorLines, errorLine)
		}
	}
	
	return errorLines
}

// isErrorLine determines if a line contains error information
func (cfa *ciFailureAnalyzer) isErrorLine(line string) bool {
	lowerLine := strings.ToLower(line)
	
	// Common error patterns
	errorPatterns := []string{
		"fail:",
		"error:",
		"panic:",
		"##[error]",
		"fatal:",
		"exception:",
		"failed",
		"assertion",
	}
	
	for _, pattern := range errorPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	
	return false
}

// determineSeverity determines the severity level of an error line
func (cfa *ciFailureAnalyzer) determineSeverity(line string) string {
	lowerLine := strings.ToLower(line)
	
	if strings.Contains(lowerLine, "fatal") || strings.Contains(lowerLine, "panic") {
		return "fatal"
	}
	if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "fail") {
		return "error"
	}
	if strings.Contains(lowerLine, "warn") {
		return "warning"
	}
	
	return "info"
}

// generateSummary generates a summary based on failure type and error lines
func (cfa *ciFailureAnalyzer) generateSummary(failureType FailureType, errorLines []ErrorLine) string {
	switch failureType {
	case TestFailure:
		return fmt.Sprintf("Test failure detected with %d error lines", len(errorLines))
	case BuildFailure:
		return fmt.Sprintf("Build failure detected with %d error lines", len(errorLines))
	case LintFailure:
		return fmt.Sprintf("Lint failure detected with %d error lines", len(errorLines))
	case TimeoutFailure:
		return "Timeout failure detected - job exceeded maximum execution time"
	default:
		return fmt.Sprintf("Unknown failure type detected with %d error lines", len(errorLines))
	}
}

// GenerateSuggestions provides suggestions based on failure analysis
func (cfa *ciFailureAnalyzer) GenerateSuggestions(analysis FailureAnalysis) []string {
	var suggestions []string
	
	switch analysis.FailureType {
	case TestFailure:
		suggestions = append(suggestions, "Review failing test cases and fix assertions")
		suggestions = append(suggestions, "Check for race conditions or timing issues")
		suggestions = append(suggestions, "Verify test data and mock configurations")
		
	case BuildFailure:
		suggestions = append(suggestions, "Check for syntax errors and compilation issues")
		suggestions = append(suggestions, "Verify import paths and dependencies")
		suggestions = append(suggestions, "Review recent code changes for breaking changes")
		
	case LintFailure:
		suggestions = append(suggestions, "Run golangci-lint locally to identify issues")
		suggestions = append(suggestions, "Fix code style and formatting issues")
		suggestions = append(suggestions, "Review linter configuration and exclusions")
		
	case TimeoutFailure:
		suggestions = append(suggestions, "Optimize slow tests or operations")
		suggestions = append(suggestions, "Consider increasing timeout limits")
		suggestions = append(suggestions, "Review for infinite loops or blocking operations")
		
	default:
		suggestions = append(suggestions, "Review CI logs for specific error messages")
		suggestions = append(suggestions, "Check system resources and dependencies")
		suggestions = append(suggestions, "Verify CI configuration and environment setup")
	}
	
	return suggestions
}

// RecordFailurePattern records a failure pattern for tracking
func (cfa *ciFailureAnalyzer) RecordFailurePattern(pattern FailurePattern) error {
	cfa.mutex.Lock()
	defer cfa.mutex.Unlock()
	
	key := fmt.Sprintf("%s:%s", pattern.Type, pattern.Pattern)
	
	if existing, exists := cfa.failurePatterns[key]; exists {
		// Update existing pattern
		existing.Frequency++
		existing.LastSeen = time.Now()
	} else {
		// Create new pattern
		pattern.LastSeen = time.Now()
		if pattern.Frequency == 0 {
			pattern.Frequency = 1
		}
		cfa.failurePatterns[key] = &pattern
	}
	
	return nil
}

// GetFailurePatterns returns all recorded failure patterns
func (cfa *ciFailureAnalyzer) GetFailurePatterns() []FailurePattern {
	cfa.mutex.RLock()
	defer cfa.mutex.RUnlock()
	
	patterns := make([]FailurePattern, 0, len(cfa.failurePatterns))
	for _, pattern := range cfa.failurePatterns {
		patterns = append(patterns, *pattern)
	}
	
	return patterns
}