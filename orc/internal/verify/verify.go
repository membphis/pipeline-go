package verify

import (
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func (r *Result) Success() bool {
	return r.ReturnCode == 0
}

var execCommand = exec.Command

func Run(spec interface{}, timeout time.Duration, workDir string) ([]Result, error) {
	switch v := spec.(type) {
	case nil:
		return nil, nil
	case string:
		r, err := runOne(v, timeout, workDir)
		if err != nil {
			return nil, err
		}
		return []Result{*r}, nil
	case []string:
		var results []Result
		for _, cmd := range v {
			r, err := runOne(cmd, timeout, workDir)
			if err != nil {
				return nil, err
			}
			results = append(results, *r)
		}
		return results, nil
	case []interface{}:
		var results []Result
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			r, err := runOne(s, timeout, workDir)
			if err != nil {
				return nil, err
			}
			results = append(results, *r)
		}
		return results, nil
	default:
		return nil, nil
	}
}

func runOne(cmdStr string, timeout time.Duration, workDir string) (*Result, error) {
	cmd := execCommand("sh", "-c", cmdStr)
	cmd.Dir = workDir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if timeout > 0 {
		if err := cmd.Start(); err != nil {
			return &Result{Stderr: err.Error()}, err
		}
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-done:
		case <-time.After(timeout):
			cmd.Process.Kill()
			<-done
		}
	} else {
		cmd.Run()
	}

	return &Result{
		ReturnCode: cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
}

func AllSuccessful(results []Result) bool {
	for _, r := range results {
		if !r.Success() {
			return false
		}
	}
	return true
}
