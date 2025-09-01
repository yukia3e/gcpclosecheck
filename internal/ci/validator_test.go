package ci

import (
	"testing"
)

func TestNewCIValidator(t *testing.T) {
	validator := NewCIValidator("test-repo", "test-token")
	if validator == nil {
		t.Fatal("Expected CIValidator instance, got nil")
	}
}

func TestCIValidator_ValidateGitHubActions(t *testing.T) {
	validator := NewCIValidator("test-repo", "test-token")

	// Test with sample commit SHA
	result, err := validator.ValidateGitHubActions("abc123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected CIResult, got nil")
	}

	// Verify result structure
	if result.CommitSHA == "" {
		t.Error("Expected CommitSHA to be set")
	}

	if result.Status == "" {
		t.Error("Expected Status to be set")
	}
}

func TestCIResult_Structure(t *testing.T) {
	jobs := []JobResult{
		{Name: "build", Status: "success", Duration: 120},
		{Name: "test", Status: "success", Duration: 180},
		{Name: "lint", Status: "failure", Duration: 60},
	}

	result := CIResult{
		CommitSHA: "abc123",
		Status:    "failure",
		Jobs:      jobs,
		Success:   false,
	}

	if result.CommitSHA != "abc123" {
		t.Error("Expected CommitSHA to be abc123")
	}

	if result.Status != "failure" {
		t.Error("Expected Status to be failure")
	}

	if len(result.Jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(result.Jobs))
	}

	if result.Success {
		t.Error("Expected Success to be false")
	}
}

func TestJobResult_Structure(t *testing.T) {
	job := JobResult{
		Name:     "build",
		Status:   "success",
		Duration: 120,
		LogURL:   "https://github.com/test/actions/runs/123",
	}

	if job.Name != "build" {
		t.Error("Expected Name to be build")
	}

	if job.Status != "success" {
		t.Error("Expected Status to be success")
	}

	if job.Duration != 120 {
		t.Error("Expected Duration to be 120")
	}

	if job.LogURL == "" {
		t.Error("Expected LogURL to be set")
	}
}

func TestCIValidator_ParseGitHubActionsResponse(t *testing.T) {
	// Sample GitHub Actions API response JSON
	sampleResponse := `{
		"workflow_runs": [
			{
				"id": 123456789,
				"head_sha": "abc123",
				"status": "completed",
				"conclusion": "failure",
				"jobs_url": "https://api.github.com/repos/test/jobs"
			}
		]
	}`

	validator := NewCIValidator("test-repo", "test-token")
	result, err := validator.ParseGitHubActionsResponse(sampleResponse)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected CIResult, got nil")
	}

	if result.CommitSHA != "abc123" {
		t.Errorf("Expected CommitSHA abc123, got %s", result.CommitSHA)
	}

	if result.Status != "failure" {
		t.Errorf("Expected Status failure, got %s", result.Status)
	}
}

func TestCIValidator_GetJobDetails(t *testing.T) {
	// Sample GitHub Actions jobs API response
	jobsResponse := `{
		"jobs": [
			{
				"name": "build",
				"status": "completed",
				"conclusion": "success",
				"started_at": "2023-01-01T12:00:00Z",
				"completed_at": "2023-01-01T12:02:00Z",
				"html_url": "https://github.com/test/actions/runs/123/jobs/456"
			},
			{
				"name": "test",
				"status": "completed", 
				"conclusion": "failure",
				"started_at": "2023-01-01T12:02:00Z",
				"completed_at": "2023-01-01T12:05:00Z",
				"html_url": "https://github.com/test/actions/runs/123/jobs/789"
			}
		]
	}`

	validator := NewCIValidator("test-repo", "test-token")
	jobs, err := validator.GetJobDetails(jobsResponse)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Verify first job
	if len(jobs) > 0 {
		job := jobs[0]
		if job.Name != "build" {
			t.Errorf("Expected job name 'build', got %s", job.Name)
		}
		if job.Status != "success" {
			t.Errorf("Expected job status 'success', got %s", job.Status)
		}
		if job.Duration == 0 {
			t.Error("Expected job duration to be calculated")
		}
	}

	// Verify second job (failure)
	if len(jobs) > 1 {
		job := jobs[1]
		if job.Name != "test" {
			t.Errorf("Expected job name 'test', got %s", job.Name)
		}
		if job.Status != "failure" {
			t.Errorf("Expected job status 'failure', got %s", job.Status)
		}
	}
}

func TestCIValidator_IsSuccessful(t *testing.T) {
	successfulJobs := []JobResult{
		{Name: "build", Status: "success"},
		{Name: "test", Status: "success"},
		{Name: "lint", Status: "success"},
	}

	failedJobs := []JobResult{
		{Name: "build", Status: "success"},
		{Name: "test", Status: "failure"},
		{Name: "lint", Status: "success"},
	}

	validator := NewCIValidator("test-repo", "test-token")

	if !validator.IsSuccessful(successfulJobs) {
		t.Error("Expected successful jobs to return true")
	}

	if validator.IsSuccessful(failedJobs) {
		t.Error("Expected failed jobs to return false")
	}
}
