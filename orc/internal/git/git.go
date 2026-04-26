package git

import (
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func run(args ...string) (string, error) {
	cmd := execCommand("git", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func Tag(name string) error {
	_, err := run("tag", name)
	return err
}
