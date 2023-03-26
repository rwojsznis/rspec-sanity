package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunnerRun(t *testing.T) {
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
