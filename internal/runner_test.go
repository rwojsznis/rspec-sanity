package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
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
	tempFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	scriptFile, err := ioutil.TempFile("", "script")
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
				Command: fmt.Sprintf("/bin/bash %s", scriptFile.Name()),
				Arguments: "1",
				RerunArguments: "0",
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
