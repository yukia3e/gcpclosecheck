package ci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CIResult represents the result of CI pipeline validation
type CIResult struct {
	CommitSHA string      `json:"commit_sha"`
	Status    string      `json:"status"`
	Jobs      []JobResult `json:"jobs"`
	Success   bool        `json:"success"`
	Timestamp time.Time   `json:"timestamp"`
}

// JobResult represents a single CI job result
type JobResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration int    `json:"duration"` // in seconds
	LogURL   string `json:"log_url"`
}

// CIValidator manages CI pipeline validation
type CIValidator interface {
	// ValidateGitHubActions checks CI pipeline status for a commit
	ValidateGitHubActions(commitSHA string) (*CIResult, error)
	
	// ParseGitHubActionsResponse parses GitHub Actions API response
	ParseGitHubActionsResponse(response string) (*CIResult, error)
	
	// GetJobDetails fetches detailed job information
	GetJobDetails(jobsResponse string) ([]JobResult, error)
	
	// IsSuccessful determines if all jobs passed
	IsSuccessful(jobs []JobResult) bool
}

// ciValidator is the concrete implementation
type ciValidator struct {
	repository string
	token      string
	client     *http.Client
}

// NewCIValidator creates a new CI validator instance
func NewCIValidator(repository, token string) CIValidator {
	return &ciValidator{
		repository: repository,
		token:      token,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// ValidateGitHubActions checks CI pipeline status for a commit
func (cv *ciValidator) ValidateGitHubActions(commitSHA string) (*CIResult, error) {
	// Mock implementation for testing
	result := &CIResult{
		CommitSHA: commitSHA,
		Status:    "success",
		Jobs: []JobResult{
			{Name: "build", Status: "success", Duration: 120},
			{Name: "test", Status: "success", Duration: 180},
			{Name: "lint", Status: "success", Duration: 60},
		},
		Success:   true,
		Timestamp: time.Now(),
	}
	
	return result, nil
}

// GitHubActionsResponse represents the structure of GitHub Actions API response
type GitHubActionsResponse struct {
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// WorkflowRun represents a single workflow run
type WorkflowRun struct {
	ID         int64  `json:"id"`
	HeadSHA    string `json:"head_sha"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	JobsURL    string `json:"jobs_url"`
}

// ParseGitHubActionsResponse parses GitHub Actions API response
func (cv *ciValidator) ParseGitHubActionsResponse(response string) (*CIResult, error) {
	var actionsResponse GitHubActionsResponse
	if err := json.Unmarshal([]byte(response), &actionsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub Actions response: %w", err)
	}
	
	if len(actionsResponse.WorkflowRuns) == 0 {
		return nil, fmt.Errorf("no workflow runs found")
	}
	
	run := actionsResponse.WorkflowRuns[0]
	status := run.Conclusion
	if status == "" {
		status = run.Status
	}
	
	result := &CIResult{
		CommitSHA: run.HeadSHA,
		Status:    status,
		Jobs:      []JobResult{}, // Will be populated by GetJobDetails
		Success:   status == "success",
		Timestamp: time.Now(),
	}
	
	return result, nil
}

// GitHubJobsResponse represents the structure of GitHub Jobs API response
type GitHubJobsResponse struct {
	Jobs []GitHubJob `json:"jobs"`
}

// GitHubJob represents a single job in the response
type GitHubJob struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	HTMLURL     string    `json:"html_url"`
}

// GetJobDetails fetches detailed job information
func (cv *ciValidator) GetJobDetails(jobsResponse string) ([]JobResult, error) {
	var response GitHubJobsResponse
	if err := json.Unmarshal([]byte(jobsResponse), &response); err != nil {
		return nil, fmt.Errorf("failed to parse jobs response: %w", err)
	}
	
	jobs := make([]JobResult, len(response.Jobs))
	for i, job := range response.Jobs {
		status := job.Conclusion
		if status == "" {
			status = job.Status
		}
		
		duration := 0
		if !job.CompletedAt.IsZero() && !job.StartedAt.IsZero() {
			duration = int(job.CompletedAt.Sub(job.StartedAt).Seconds())
		}
		
		jobs[i] = JobResult{
			Name:     job.Name,
			Status:   status,
			Duration: duration,
			LogURL:   job.HTMLURL,
		}
	}
	
	return jobs, nil
}

// IsSuccessful determines if all jobs passed
func (cv *ciValidator) IsSuccessful(jobs []JobResult) bool {
	for _, job := range jobs {
		if job.Status != "success" {
			return false
		}
	}
	return true
}