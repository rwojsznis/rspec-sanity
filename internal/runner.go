package internal

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	Settings *Settings
}

type RunnerResult struct {
	StatusCode    int
	Error         error
	FlakyExamples []RspecExample
}

func (rr *RunnerResult) HasFlakies() bool {
	return len(rr.FlakyExamples) > 0
}

func (r *Runner) Run() RunnerResult {
	command := r.Settings.Config.RunCommand(r.Settings.Pattern)
	status, err := r.exec(command, 1)

	if status == 0 {
		log.Println("[rspec-sanity] Build succeeded at first attempt")
		return RunnerResult{
			StatusCode: status,
			Error:      err,
		}
	} else if r.Settings.SkipRerun {
		log.Printf("[rspec-sanity] Build failed with %v, but skipping rerun", err)
		return RunnerResult{
			StatusCode: status,
			Error:      err,
		}
	} else {
		log.Println("[rspec-sanity] Build failed, rerunning failed tests")

		examplesFirstRun, err := r.Settings.Config.CollectExamples()

		if err != nil {
			return RunnerResult{
				StatusCode: status,
				Error:      err,
			}
		}

		command = r.Settings.Config.RerunCommand(r.Settings.Pattern)
		status, err = r.exec(command, 2)

		examplesSecondRun, err := r.Settings.Config.CollectExamples()
		flakies := FindFlakies(examplesFirstRun, examplesSecondRun)

		return RunnerResult{
			StatusCode: status,
			Error:      err,
			FlakyExamples: flakies,
		}
	}
}

func (r *Runner) exec(command string, attempt int) (int, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	args := strings.Fields(command)

	log.Println("[rspec-sanity] Running external command: ", args)

	cmd := exec.Command(
		args[0],
		args[1:]...,
	)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("RSPEC_SANITY_ATTEMPT=%d", attempt))

	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Start()

	if err != nil {
		return 1, err
	}

	err = cmd.Wait()

	if exiterr, ok := err.(*exec.ExitError); ok {
		return exiterr.ExitCode(), err
	} else if err != nil {
		return 1, err
	} else {
		return 0, nil
	}
}
