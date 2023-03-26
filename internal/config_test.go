package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	_, err := LoadConfig("invalid")
	assert.Error(t, err)
	assert.Equal(t, `error reading config file from: "invalid" ("stat invalid: no such file or directory")`, err.Error())

	tempFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)

	defer os.Remove(tempFile.Name())

	data := `
command = "bundle exec rspec"
arguments = "--format documentation --force-color"
rerun_arguments = "--format progress"
persistence_file = "spec/examples.txt"
`
	_, err = tempFile.Write([]byte(data))
	assert.NoError(t, err)

	config, err := LoadConfig(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "bundle exec rspec", config.Command)
	assert.Equal(t, "--format documentation --force-color", config.Arguments)
	assert.Equal(t, "--format progress", config.RerunArguments)
	assert.Equal(t, "spec/examples.txt", config.PersistenceFile)
}

func TestRunCommand(t *testing.T) {
	config := Config{
		Command:   "bundle exec rspec",
		Arguments: "--format documentation",
	}

	assert.Equal(
		t,
		"bundle exec rspec --format documentation spec/",
		config.RunCommand([]string{"spec/"}),
	)

	config = Config{
		Command:   "rspec",
		Arguments: "",
	}

	assert.Equal(
		t,
		"rspec spec/lib spec/models",
		config.RunCommand([]string{"spec/lib spec/models"}),
	)
}

func TestRerunCommand(t *testing.T) {
	config := Config{
		Command:        "bundle exec rspec",
		Arguments:      "--format documentation",
		RerunArguments: "",
	}

	assert.Equal(
		t,
		"bundle exec rspec --only-failures spec/",
		config.RerunCommand([]string{"spec/"}),
	)

	config = Config{
		Command:        "bin/rspec",
		RerunArguments: "--format documentation",
	}

	assert.Equal(
		t,
		"bin/rspec --format documentation --only-failures spec/",
		config.RerunCommand([]string{"spec/"}),
	)
}

func TestRenderTemplate(t *testing.T) {
	examples := []RspecExample{
		{Id: "foo"},
		{Id: "bar"},
	}

	template := `Hello {{ .Env.USER }}{{ range .Examples }}
| {{ .Id }} |{{end}}`

	expected := fmt.Sprintf("Hello %s\n| foo |\n| bar |", os.Getenv("USER"))

	result, err := RenderTemplate(template, examples)

	assert.NoError(t, err)
	assert.Equal(
		t,
		expected,
		result,
	)
}
