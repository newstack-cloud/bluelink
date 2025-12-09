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

type gitImpl struct {
	gitPath string
}

// NewDefaultGit creates a new instance of the default Git implementation.
// It looks up the git binary path at construction time.
func NewDefaultGit() Git {
	// Look up git path - if not found, store empty string and error at execution time
	gitPath, _ := exec.LookPath("git")
	return &gitImpl{gitPath: gitPath}
}

// Clones a git repository to a specified directory.
func (g *gitImpl) Clone(repoURL string, directory string) error {
	if g.gitPath == "" {
		return fmt.Errorf("git executable not found in PATH")
	}
	cmd := exec.Command(g.gitPath, "clone", repoURL, directory)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}
	return nil
}

// Initializes a new git repository in the specified directory.
func (g *gitImpl) Init(directory string) error {
	if g.gitPath == "" {
		return fmt.Errorf("git executable not found in PATH")
	}
	cmd := exec.Command(g.gitPath, "init")
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\n%s", err, string(output))
	}
	return nil
}
