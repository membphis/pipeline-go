package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func runInDir(repoPath string, args ...string) (string, error) {
	cmd := execCommand("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func RepoInit(repoPath string) error {
	_, err := runInDir(repoPath, "init")
	return err
}

func InitCommit(repoPath string) error {
	readmePath := repoPath + "/README"
	if err := os.WriteFile(readmePath, []byte{}, 0644); err != nil {
		return fmt.Errorf("writing README: %w", err)
	}
	if _, err := runInDir(repoPath, "add", "README"); err != nil {
		return err
	}
	if _, err := runInDir(repoPath, "-c", "user.name=orc", "-c", "user.email=orc@local", "commit", "-m", "init"); err != nil {
		return err
	}
	return nil
}

func Tag(repoPath, name string) error {
	_, err := runInDir(repoPath, "tag", name)
	return err
}
