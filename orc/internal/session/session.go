package session

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var execCommand = exec.Command

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func Run(prompt string, timeout time.Duration) (*Result, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("opencode not found on $PATH: install via 'pip install opencode' or download from https://opencode.ai")
	}
	fmt.Fprintf(os.Stderr, "command: opencode run --dangerously-skip-permissions (prompt=%d bytes)\n", len(prompt))

	cmd := execCommand(path, "run", "--dangerously-skip-permissions", prompt)
	cmd.Stdin = os.Stdin

	var stdout, stderr strings.Builder
	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting opencode: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if timeout > 0 {
		select {
		case <-done:
		case <-time.After(timeout):
			cmd.Process.Kill()
			<-done
			return &Result{
				ReturnCode: -9,
				Stdout:     stdout.String(),
				Stderr:     stderr.String(),
			}, nil
		}
	} else {
		<-done
	}

	return &Result{
		ReturnCode: cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
}
