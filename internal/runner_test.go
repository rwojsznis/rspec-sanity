package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnerFirstRun(t *testing.T) {
	runner := &Runner{
		Settings: &Settings{
			Config: Config{
				Command: "echo 'hello world'",
			},
		},
	}

	result := runner.Run()

	assert.Nil(t, result.Error)
	assert.Equal(t, 0, result.StatusCode)
}

func TestRunnerSecondRun(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	scriptFile, err := os.CreateTemp("", "script")
	assert.NoError(t, err)
	defer os.Remove(scriptFile.Name())

	data := `#!/bin/bash
if [ "$1" == "1" ]; then
	exit 1
fi

exit 0
`
	_, err = scriptFile.Write([]byte(data))
	assert.NoError(t, err)

	runner := &Runner{
		Settings: &Settings{
			Config: Config{
				PersistenceFile: tempFile.Name(),
				Command:         fmt.Sprintf("/bin/bash %s", scriptFile.Name()),
				Arguments:       "1",
				RerunArguments:  "0",
			},
		},
	}

	result := runner.Run()

	assert.Nil(t, result.Error)
	assert.Equal(t, 0, result.StatusCode)

	runner.Settings.Config.RerunArguments = "1"
	result = runner.Run()
	assert.Error(t, &exec.ExitError{}, result.Error)
	assert.Equal(t, 1, result.StatusCode)
}

func TestRunnerDetectsFlakyExamplesAcrossAttempts(t *testing.T) {
	dir := t.TempDir()
	persistenceFile := fmt.Sprintf("%s/examples.txt", dir)
	scriptFile := fmt.Sprintf("%s/rspec", dir)
	script := `#!/bin/sh
cat > "$1" <<EOF
example_id | status | run_time |
-----------|--------|----------|
spec/flaky_spec.rb[1:1] | $([ "$RSPEC_SANITY_ATTEMPT" = 1 ] && printf failed || printf passed) | 0.1 |
spec/stable_spec.rb[1:1] | passed | 0.1 |
EOF
[ "$RSPEC_SANITY_ATTEMPT" = 1 ] && exit 1
exit 0
`
	assert.NoError(t, os.WriteFile(scriptFile, []byte(script), 0o700))

	runner := &Runner{Settings: &Settings{Config: Config{
		Command:         scriptFile,
		Arguments:       persistenceFile,
		RerunArguments:  persistenceFile,
		PersistenceFile: persistenceFile,
	}}}

	result := runner.Run()
	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.StatusCode)
	assert.Equal(t, []RspecExample{{Id: "spec/flaky_spec.rb[1:1]", Status: "failed"}}, result.FlakyExamples)
	assert.True(t, result.HasFlakies())
}

func TestRunnerCanSkipRerun(t *testing.T) {
	runner := &Runner{Settings: &Settings{
		SkipRerun: true,
		Config:    Config{Command: "/bin/sh -c false"},
	}}

	result := runner.Run()
	assert.Equal(t, 1, result.StatusCode)
	assert.Error(t, result.Error)
	assert.False(t, result.HasFlakies())
}

func TestRunnerReturnsPersistenceErrorAfterFailedFirstAttempt(t *testing.T) {
	command, err := exec.LookPath("false")
	require.NoError(t, err)
	runner := &Runner{Settings: &Settings{Config: Config{
		Command:         command,
		PersistenceFile: fmt.Sprintf("%s/missing.txt", t.TempDir()),
	}}}

	result := runner.Run()
	assert.Equal(t, 1, result.StatusCode)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "missing.txt")
}

func TestRunnerReturnsCommandStartupError(t *testing.T) {
	runner := &Runner{Settings: &Settings{
		SkipRerun: true,
		Config:    Config{Command: filepath.Join(t.TempDir(), "missing-rspec")},
	}}

	result := runner.Run()
	assert.Equal(t, 1, result.StatusCode)
	assert.Error(t, result.Error)
}
