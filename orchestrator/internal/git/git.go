package git

import (
	"fmt"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func run(args ...string) (string, error) {
	cmd := execCommand("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func CurrentBranch() (string, error) {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", fmt.Errorf("detached HEAD state")
	}
	return out, nil
}

func CreateBranch(name string, base ...string) error {
	args := []string{"checkout", "-b", name}
	if len(base) > 0 && base[0] != "" {
		args = append(args, base[0])
	}
	_, err := run(args...)
	return err
}

func Commit(message string) error {
	_, err := run("commit", "-m", message)
	return err
}

func Checkout(branch string) error {
	_, err := run("checkout", branch)
	return err
}

func SquashMerge(branch string) error {
	if _, err := run("merge", "--squash", branch); err != nil {
		return err
	}
	_, err := run("commit", "-m", fmt.Sprintf("Squash merge %s", branch))
	return err
}

func Tag(name string, message ...string) error {
	if len(message) > 0 && message[0] != "" {
		_, err := run("tag", "-a", name, "-m", message[0])
		return err
	}
	_, err := run("tag", name)
	return err
}

func IsClean() bool {
	out, err := run("status", "--porcelain")
	return err == nil && out == ""
}

func HasUnpushedCommits() bool {
	out, err := run("rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		return false
	}
	return out != "0"
}

func IsDetachedHead() bool {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	return err == nil && out == "HEAD"
}
