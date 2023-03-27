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

func TestLoadConfigWithGithub(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	data := `
command = "bundle exec rspec"
arguments = "--format documentation --force-color"
rerun_arguments = "--format progress"
persistence_file = "spec/examples.txt"

[github]
owner = "jdoe"
repo = "rspec-sanity"
labels = ['flaky-spec']
template = 'template string'
`

	_, err = tempFile.Write([]byte(data))
	assert.NoError(t, err)

	originalToken := os.Getenv("RSPEC_SANITY_GITHUB_TOKEN")
	os.Setenv("RSPEC_SANITY_GITHUB_TOKEN", "my-gh-token")

	defer func() {
		os.Setenv("RSPEC_SANITY_GITHUB_TOKEN", originalToken)
	}()

	config, err := LoadConfig(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []string{"flaky-spec"}, config.Github.Labels)
	assert.Equal(t, "template string", config.Github.Template)
	assert.Equal(t, "my-gh-token", config.Github.GetToken())
	assert.Equal(t, "jdoe", config.Github.Owner)
	assert.Equal(t, "rspec-sanity", config.Github.Repo)
}

func TestLoadConfigWithJira(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	data := `
command = "bundle exec rspec"
arguments = "--format documentation --force-color"
rerun_arguments = "--format progress"
persistence_file = "spec/examples.txt"

[jira]
epic_id = "PROD-1"
task_type_id = "10001"
project_id = "PROD"
labels = ['flaky-spec-label']
template = 'template string'
`

	_, err = tempFile.Write([]byte(data))
	assert.NoError(t, err)

	originalToken := os.Getenv("RSPEC_SANITY_JIRA_TOKEN")
	originalUser := os.Getenv("RSPEC_SANITY_JIRA_USER")
	orginalHost := os.Getenv("RSPEC_SANITY_JIRA_HOST")

	os.Setenv("RSPEC_SANITY_JIRA_TOKEN", "my-jira-token")
	os.Setenv("RSPEC_SANITY_JIRA_USER", "my-jira-user")
	os.Setenv("RSPEC_SANITY_JIRA_HOST", "my-jira-host")

	defer func() {
		os.Setenv("RSPEC_SANITY_JIRA_TOKEN", originalToken)
		os.Setenv("RSPEC_SANITY_JIRA_USER", originalUser)
		os.Setenv("RSPEC_SANITY_JIRA_HOST", orginalHost)
	}()

	config, err := LoadConfig(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []string{"flaky-spec-label"}, config.Jira.Labels)
	assert.Equal(t, "template string", config.Jira.Template)
	assert.Equal(t, "my-jira-token", config.Jira.GetToken())
	assert.Equal(t, "my-jira-user", config.Jira.GetUser())
	assert.Equal(t, "my-jira-host", config.Jira.GetHost())
	assert.Equal(t, "PROD-1", config.Jira.EpicId)
	assert.Equal(t, "10001", config.Jira.TaskTypeId)
	assert.Equal(t, "PROD", config.Jira.ProjectId)
}

func TestGetReporter(t *testing.T) {
	config := Config{}
	assert.Equal(t, &NullReporter{}, config.GetReporter())

	config.Github = &GithubConfig{}
	assert.Equal(t, NewGithubReporter(config.Github), config.GetReporter())

	config.Github = nil
	config.Jira = &JiraConfig{}
	assert.Equal(t, NewJiraReporter(config.Jira), config.GetReporter())
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

func TestCollectExamples(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	data := `example_id                       | status | run_time        |
-------------------------------- | ------ | --------------- |
./spec/flaky_spec.rb[1:1]        | passed | 0.00051 seconds |
./spec/flaky_spec.rb[1:2]        | passed | 0.00005 seconds |
./spec/flaky_spec.rb[1:3]        | passed | 0.00004 seconds |
./spec/new_flaky_spec.rb[1:1]    | failed | 0.00004 seconds |

`
	_, err = tempFile.Write([]byte(data))
	assert.NoError(t, err)

	config := Config{PersistenceFile: tempFile.Name()}

	examples, err := config.CollectExamples()
	assert.NoError(t, err)
	assert.Equal(t, 4, len(examples))

	assert.Equal(t, "./spec/flaky_spec.rb[1:1]", examples[0].Id)
	assert.Equal(t, "passed", examples[1].Status)
	assert.Equal(t, "./spec/new_flaky_spec.rb[1:1]", examples[3].Id)
	assert.Equal(t, "failed", examples[3].Status)
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
