package ci

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitCommit represents a git commit
type GitCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// GitStatus represents the current git repository status
type GitStatus struct {
	ModifiedFiles  []string `json:"modified_files"`
	UntrackedFiles []string `json:"untracked_files"`
	StagedFiles    []string `json:"staged_files"`
	Branch         string   `json:"branch"`
	Clean          bool     `json:"clean"`
}

// GitManager manages git operations
type GitManager interface {
	// InitRepository initializes a git repository
	InitRepository() error

	// CreateCommit creates a commit with standardized commit message
	CreateCommit(message string) (string, error)

	// PushChanges pushes changes to remote with error handling and retry logic
	PushChanges(branch string) error

	// CreateBranch creates and switches to a new branch
	CreateBranch(branchName string) error

	// GetCurrentBranch returns the current branch name
	GetCurrentBranch() (string, error)

	// AddFiles adds files to staging area
	AddFiles(files []string) error

	// GetCommitHistory returns recent commit history
	GetCommitHistory(limit int) ([]GitCommit, error)

	// GetStatus returns current repository status
	GetStatus() (*GitStatus, error)
}

// gitManager is the concrete implementation
type gitManager struct {
	repoPath string
}

// NewGitManager creates a new git manager instance
func NewGitManager(repoPath string) GitManager {
	return &gitManager{
		repoPath: repoPath,
	}
}

// InitRepository initializes a git repository
func (gm *gitManager) InitRepository() error {
	cmd := exec.Command("git", "init")
	cmd.Dir = gm.repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Set default user config for testing
	err := gm.runGitCommand("config", "user.email", "test@example.com")
	if err != nil {
		return fmt.Errorf("failed to set git user email: %w", err)
	}

	err = gm.runGitCommand("config", "user.name", "Test User")
	if err != nil {
		return fmt.Errorf("failed to set git user name: %w", err)
	}

	return nil
}

// runGitCommand executes a git command in the repository directory
func (gm *gitManager) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = gm.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %s, output: %s", err, string(output))
	}

	return nil
}

// runGitCommandWithOutput executes a git command and returns output
func (gm *gitManager) runGitCommandWithOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = gm.repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CreateCommit creates a commit with standardized commit message
func (gm *gitManager) CreateCommit(message string) (string, error) {
	// Add timestamp and standard format to commit message
	standardizedMessage := fmt.Sprintf("%s\n\n🤖 Generated with [Claude Code](https://claude.ai/code)\n\nCo-Authored-By: Claude <noreply@anthropic.com>", message)

	err := gm.runGitCommand("commit", "-m", standardizedMessage)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Get the commit SHA
	commitSHA, err := gm.runGitCommandWithOutput("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return commitSHA, nil
}

// PushChanges pushes changes to remote with error handling and retry logic
func (gm *gitManager) PushChanges(branch string) error {
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := gm.runGitCommand("push", "origin", branch)
		if err == nil {
			return nil
		}

		// If it's the last attempt, return the error
		if attempt == maxRetries-1 {
			return fmt.Errorf("failed to push changes after %d attempts: %w", maxRetries, err)
		}

		// Wait before retry
		time.Sleep(time.Second * time.Duration(attempt+1))
	}

	return nil
}

// CreateBranch creates and switches to a new branch
func (gm *gitManager) CreateBranch(branchName string) error {
	err := gm.runGitCommand("checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and switch to branch %s: %w", branchName, err)
	}

	return nil
}

// GetCurrentBranch returns the current branch name
func (gm *gitManager) GetCurrentBranch() (string, error) {
	branch, err := gm.runGitCommandWithOutput("branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return branch, nil
}

// AddFiles adds files to staging area
func (gm *gitManager) AddFiles(files []string) error {
	args := append([]string{"add"}, files...)
	err := gm.runGitCommand(args...)
	if err != nil {
		return fmt.Errorf("failed to add files %v: %w", files, err)
	}

	return nil
}

// GetCommitHistory returns recent commit history
func (gm *gitManager) GetCommitHistory(limit int) ([]GitCommit, error) {
	limitStr := strconv.Itoa(limit)
	output, err := gm.runGitCommandWithOutput("log", "--oneline", "--pretty=format:%H|%s|%an|%ae|%ct", "-n", limitStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	if output == "" {
		return []GitCommit{}, nil
	}

	lines := strings.Split(output, "\n")
	commits := make([]GitCommit, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}

		timestampInt, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			continue
		}

		commit := GitCommit{
			SHA:       parts[0],
			Message:   parts[1],
			Author:    parts[2],
			Email:     parts[3],
			Timestamp: time.Unix(timestampInt, 0),
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// GetStatus returns current repository status
func (gm *gitManager) GetStatus() (*GitStatus, error) {
	// Get current branch
	branch, err := gm.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get status output
	output, err := gm.runGitCommandWithOutput("status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	status := &GitStatus{
		ModifiedFiles:  []string{},
		UntrackedFiles: []string{},
		StagedFiles:    []string{},
		Branch:         branch,
		Clean:          output == "",
	}

	if output == "" {
		return status, nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		filename := line[3:]

		switch statusCode {
		case "??":
			status.UntrackedFiles = append(status.UntrackedFiles, filename)
		case " M", "MM":
			status.ModifiedFiles = append(status.ModifiedFiles, filename)
		case "A ", "M ", "D ":
			status.StagedFiles = append(status.StagedFiles, filename)
		}
	}

	return status, nil
}

// ensureDirectory creates directory if it doesn't exist
