package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var execCommand = exec.Command

func RunInDir(repoPath string, args ...string) (string, error) {
	cmd := execCommand("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return result, fmt.Errorf("git %s: %s", strings.Join(args, " "), result)
	}
	return result, nil
}

func RepoInit(repoPath string) error {
	_, err := RunInDir(repoPath, "init")
	return err
}

func InitCommit(repoPath string) error {
	readmePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte{}, 0644); err != nil {
		return fmt.Errorf("writing README.md: %w", err)
	}

	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(".orc_history/\n"), 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	orcHistDir := filepath.Join(repoPath, ".orc_history")
	if err := os.MkdirAll(orcHistDir, 0755); err != nil {
		return fmt.Errorf("creating .orc_history: %w", err)
	}
	orcGitignorePath := filepath.Join(orcHistDir, ".gitignore")
	if err := os.WriteFile(orcGitignorePath, []byte("*\n"), 0644); err != nil {
		return fmt.Errorf("writing .orc_history/.gitignore: %w", err)
	}

	if _, err := RunInDir(repoPath, "add", "README.md", ".gitignore"); err != nil {
		return err
	}
	// Force-add .orc_history/.gitignore since .orc_history/ is in .gitignore
	if _, err := RunInDir(repoPath, "add", "-f", ".orc_history/.gitignore"); err != nil {
		return err
	}
	if _, err := RunInDir(repoPath, "-c", "user.name=orc", "-c", "user.email=orc@local", "commit", "-m", "init"); err != nil {
		return err
	}
	return nil
}

func Tag(repoPath, name string) error {
	_, err := RunInDir(repoPath, "tag", name)
	return err
}
