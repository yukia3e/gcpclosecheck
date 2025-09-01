package ci

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGitManager(t *testing.T) {
	manager := NewGitManager("/test/repo")
	if manager == nil {
		t.Fatal("Expected GitManager instance, got nil")
	}
}

func TestGitManager_CreateCommit(t *testing.T) {
	// Create temporary directory for git operations
	tmpDir := t.TempDir()

	manager := NewGitManager(tmpDir)

	// Initialize git repository
	err := manager.InitRepository()
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add file to staging
	err = manager.AddFiles([]string{"test.txt"})
	if err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	// Create commit
	commitSHA, err := manager.CreateCommit("Test commit message")
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	if commitSHA == "" {
		t.Error("Expected commit SHA to be returned")
	}

	if len(commitSHA) < 7 {
		t.Error("Expected commit SHA to be at least 7 characters")
	}
}

func TestGitManager_PushChanges(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewGitManager(tmpDir)

	// Test push without remote (should handle gracefully)
	err := manager.PushChanges("main")
	if err == nil {
		t.Error("Expected error when pushing without remote")
	}
}

func TestGitManager_CreateBranch(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewGitManager(tmpDir)

	// Initialize repository
	err := manager.InitRepository()
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create initial commit (required for branch creation)
	testFile := filepath.Join(tmpDir, "initial.txt")
	err = os.WriteFile(testFile, []byte("initial"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	err = manager.AddFiles([]string{"initial.txt"})
	if err != nil {
		t.Fatalf("Failed to add initial file: %v", err)
	}

	_, err = manager.CreateCommit("Initial commit")
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Create new branch
	err = manager.CreateBranch("feature-branch")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Verify current branch
	currentBranch, err := manager.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if currentBranch != "feature-branch" {
		t.Errorf("Expected current branch 'feature-branch', got %s", currentBranch)
	}
}

func TestGitManager_GetCommitHistory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewGitManager(tmpDir)

	// Initialize repository and create commits
	err := manager.InitRepository()
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create first commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	err = os.WriteFile(file1, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = manager.AddFiles([]string{"file1.txt"})
	if err != nil {
		t.Fatalf("Failed to add file1: %v", err)
	}

	commit1, err := manager.CreateCommit("First commit")
	if err != nil {
		t.Fatalf("Failed to create first commit: %v", err)
	}

	// Create second commit
	file2 := filepath.Join(tmpDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	err = manager.AddFiles([]string{"file2.txt"})
	if err != nil {
		t.Fatalf("Failed to add file2: %v", err)
	}

	commit2, err := manager.CreateCommit("Second commit")
	if err != nil {
		t.Fatalf("Failed to create second commit: %v", err)
	}

	// Get commit history
	commits, err := manager.GetCommitHistory(5)
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}

	// Verify commits are in reverse chronological order
	if len(commits) >= 2 {
		if commits[0].SHA != commit2 {
			t.Errorf("Expected first commit to be %s, got %s", commit2, commits[0].SHA)
		}
		if commits[1].SHA != commit1 {
			t.Errorf("Expected second commit to be %s, got %s", commit1, commits[1].SHA)
		}
	}
}

func TestGitCommit_Structure(t *testing.T) {
	commit := GitCommit{
		SHA:       "abc123",
		Message:   "Test commit",
		Author:    "Test Author",
		Email:     "test@example.com",
		Timestamp: time.Now(),
	}

	if commit.SHA != "abc123" {
		t.Error("Expected SHA to be abc123")
	}

	if commit.Message != "Test commit" {
		t.Error("Expected Message to be 'Test commit'")
	}

	if commit.Author != "Test Author" {
		t.Error("Expected Author to be 'Test Author'")
	}

	if commit.Email != "test@example.com" {
		t.Error("Expected Email to be 'test@example.com'")
	}
}

func TestGitManager_GetStatus(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewGitManager(tmpDir)

	// Initialize repository
	err := manager.InitRepository()
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create and modify files to test status
	testFile := filepath.Join(tmpDir, "status_test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get status
	status, err := manager.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status == nil {
		t.Fatal("Expected GitStatus, got nil")
	}

	// Should have untracked files
	if len(status.UntrackedFiles) == 0 {
		t.Error("Expected untracked files")
	}
}

func TestGitStatus_Structure(t *testing.T) {
	status := GitStatus{
		ModifiedFiles:  []string{"modified.go"},
		UntrackedFiles: []string{"new.go"},
		StagedFiles:    []string{"staged.go"},
		Branch:         "main",
		Clean:          false,
	}

	if len(status.ModifiedFiles) != 1 {
		t.Error("Expected 1 modified file")
	}

	if len(status.UntrackedFiles) != 1 {
		t.Error("Expected 1 untracked file")
	}

	if len(status.StagedFiles) != 1 {
		t.Error("Expected 1 staged file")
	}

	if status.Branch != "main" {
		t.Error("Expected Branch to be main")
	}

	if status.Clean {
		t.Error("Expected Clean to be false")
	}
}
