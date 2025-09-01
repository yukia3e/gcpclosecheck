package issues

import (
	"fmt"
	"time"
	
	"github.com/yukia3e/gcpclosecheck/internal/validation"
)

// ResolutionStep represents a single step in the issue resolution process
type ResolutionStep struct {
	Issue    Issue     `json:"issue"`
	Action   string    `json:"action"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// ResolutionResult represents the overall result of issue resolution
type ResolutionResult struct {
	TotalIssues     int              `json:"total_issues"`
	ProcessedIssues int              `json:"processed_issues"`
	FixedIssues     int              `json:"fixed_issues"`
	FailedIssues    int              `json:"failed_issues"`
	Steps           []ResolutionStep `json:"steps"`
	Success         bool             `json:"success"`
	Duration        time.Duration    `json:"duration"`
}

// ResolutionWorkflow manages step-by-step issue resolution
type ResolutionWorkflow interface {
	// ExecuteResolution applies fixes incrementally with validation
	ExecuteResolution(issues []Issue) (*ResolutionResult, error)
	
	// ValidateStep runs validation after each fix application
	ValidateStep() error
}

// resolutionWorkflow is the concrete implementation
type resolutionWorkflow struct {
	workDir  string
	detector IssueDetector
}

// NewResolutionWorkflow creates a new resolution workflow instance
func NewResolutionWorkflow(workDir string) ResolutionWorkflow {
	return &resolutionWorkflow{
		workDir:  workDir,
		detector: NewIssueDetector(workDir),
	}
}

// ExecuteResolution applies fixes incrementally with validation
func (rw *resolutionWorkflow) ExecuteResolution(issues []Issue) (*ResolutionResult, error) {
	start := time.Now()
	
	result := &ResolutionResult{
		TotalIssues: len(issues),
		Steps:       make([]ResolutionStep, 0, len(issues)),
	}
	
	// Generate fix suggestions for all issues
	suggestions, err := rw.detector.GenerateFixSuggestions(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fix suggestions: %w", err)
	}
	
	// Process each suggestion incrementally
	for _, suggestion := range suggestions {
		stepStart := time.Now()
		step := ResolutionStep{
			Issue:  suggestion.Issue,
			Action: suggestion.Action,
		}
		
		result.ProcessedIssues++
		
		// Try to apply auto-fix if possible
		if suggestion.AutoFixable {
			err := rw.detector.ApplyAutoFix(suggestion)
			if err != nil {
				step.Success = false
				step.Error = err.Error()
				result.FailedIssues++
			} else {
				step.Success = true
				result.FixedIssues++
				
				// Validate after applying fix
				if validateErr := rw.ValidateStep(); validateErr != nil {
					step.Success = false
					step.Error = fmt.Sprintf("validation failed after fix: %v", validateErr)
					result.FailedIssues++
					result.FixedIssues--
				}
			}
		} else {
			// Mark as manual review required
			step.Success = false
			step.Error = "manual review required"
			result.FailedIssues++
		}
		
		step.Duration = time.Since(stepStart)
		result.Steps = append(result.Steps, step)
	}
	
	result.Duration = time.Since(start)
	result.Success = result.FailedIssues == 0
	
	return result, nil
}

// ValidateStep runs validation after each fix application
func (rw *resolutionWorkflow) ValidateStep() error {
	// Run a quick build check to ensure no regressions
	pipeline := validation.NewValidationPipeline(rw.workDir)
	buildResult, err := pipeline.RunBuild()
	if err != nil {
		return fmt.Errorf("validation build execution failed: %w", err)
	}
	
	if !buildResult.Success {
		return fmt.Errorf("build failed after fix application: %s", buildResult.Error)
	}
	
	return nil
}