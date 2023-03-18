package internal

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
