package session

import (
	"fmt"
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
	cmd := execCommand(path, prompt)
	cmd.Stdin = os.Stdin

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

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
