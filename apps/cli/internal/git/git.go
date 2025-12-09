package git

import (
	"fmt"
	"os/exec"
)

// Git is an interface that provides methods for interacting with git repositories.
type Git interface {
	// Clones a git repository to a specified directory.
	Clone(repoURL string, directory string) error
	// Initializes a new git repository in the specified directory.
	Init(directory string) error
}

type gitImpl struct{}

// NewGitImpl creates a new instance of the GitImpl.
func NewDefaultGit() Git {
	return &gitImpl{}
}

// Clones a git repository to a specified directory.
func (g *gitImpl) Clone(repoURL string, directory string) error {
	cmd := exec.Command("git", "clone", repoURL, directory)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}
	return nil
}

// Initializes a new git repository in the specified directory.
func (g *gitImpl) Init(directory string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\n%s", err, string(output))
	}
	return nil
}
