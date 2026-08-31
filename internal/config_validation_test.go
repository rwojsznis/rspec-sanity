package internal

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestLoadConfigRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{name: "invalid TOML", config: `command = "rspec`, wantErr: "unexpected EOF"},
		{name: "missing command", config: `persistence_file = "examples.txt"`, wantErr: "no rspec command specified"},
		{name: "missing persistence file", config: `command = "rspec"`, wantErr: "no persistence file specified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte(tt.config), 0o600))

			_, err := LoadConfig(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestReporterConfigValidation(t *testing.T) {
	t.Setenv("RSPEC_SANITY_GITHUB_TOKEN", "token")
	t.Setenv("RSPEC_SANITY_JIRA_TOKEN", "token")
	t.Setenv("RSPEC_SANITY_JIRA_USER", "user")
	t.Setenv("RSPEC_SANITY_JIRA_HOST", "https://jira.example.com")

	githubTests := []struct {
		name    string
		config  GithubConfig
		wantErr string
	}{
		{name: "owner", config: GithubConfig{}, wantErr: "no github owner"},
		{name: "repo", config: GithubConfig{Owner: "owner"}, wantErr: "no github repo"},
		{name: "template", config: GithubConfig{Owner: "owner", Repo: "repo"}, wantErr: "no github template"},
	}
	for _, tt := range githubTests {
		t.Run("github missing "+tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.config.Prepare(), tt.wantErr+" specified in config")
		})
	}

	jiraTests := []struct {
		name    string
		config  JiraConfig
		wantErr string
	}{
		{name: "epic", config: JiraConfig{}, wantErr: "no jira epic id specified in config"},
		{name: "project", config: JiraConfig{EpicId: "EPIC-1"}, wantErr: "no jira project id specified in config"},
		{name: "task type", config: JiraConfig{EpicId: "EPIC-1", ProjectId: "APP"}, wantErr: "no jira task type id specified in config"},
		{name: "template", config: JiraConfig{EpicId: "EPIC-1", ProjectId: "APP", TaskTypeId: "10001"}, wantErr: "no jira template specified in config"},
	}
	for _, tt := range jiraTests {
		t.Run("jira missing "+tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.config.Prepare(), tt.wantErr)
		})
	}
}

func TestGithubConfigRequiresToken(t *testing.T) {
	t.Setenv("RSPEC_SANITY_GITHUB_TOKEN", "temporary")
	require.NoError(t, os.Unsetenv("RSPEC_SANITY_GITHUB_TOKEN"))
	config := GithubConfig{Owner: "owner", Repo: "repo", Template: "template"}

	assert.EqualError(t, config.Prepare(), "specify github token under RSPEC_SANITY_GITHUB_TOKEN env")
}

func TestJiraConfigRequiresCredentials(t *testing.T) {
	valid := JiraConfig{EpicId: "EPIC-1", ProjectId: "APP", TaskTypeId: "10001", Template: "template"}

	t.Run("token", func(t *testing.T) {
		t.Setenv("RSPEC_SANITY_JIRA_TOKEN", "temporary")
		require.NoError(t, os.Unsetenv("RSPEC_SANITY_JIRA_TOKEN"))
		assert.EqualError(t, valid.Prepare(), "specify jira token under RSPEC_SANITY_JIRA_TOKEN env")
	})

	t.Run("user", func(t *testing.T) {
		t.Setenv("RSPEC_SANITY_JIRA_TOKEN", "token")
		t.Setenv("RSPEC_SANITY_JIRA_USER", "temporary")
		require.NoError(t, os.Unsetenv("RSPEC_SANITY_JIRA_USER"))
		assert.EqualError(t, valid.Prepare(), "specify jira user under RSPEC_SANITY_JIRA_USER env")
	})

	t.Run("host", func(t *testing.T) {
		t.Setenv("RSPEC_SANITY_JIRA_TOKEN", "token")
		t.Setenv("RSPEC_SANITY_JIRA_USER", "user")
		t.Setenv("RSPEC_SANITY_JIRA_HOST", "temporary")
		require.NoError(t, os.Unsetenv("RSPEC_SANITY_JIRA_HOST"))
		assert.EqualError(t, valid.Prepare(), "specify jira full host (including scheme) under RSPEC_SANITY_JIRA_HOST env")
	})
}

func TestSettingsLoadAndValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("command = \"rspec\"\npersistence_file = \"examples.txt\"\n"), 0o600))

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	require.NoError(t, set.Parse([]string{"spec/models", "spec/services"}))
	settings := Settings{ConfigPath: path}

	require.NoError(t, settings.Load(cli.NewContext(nil, set, nil)))
	assert.Equal(t, []string{"spec/models", "spec/services"}, settings.Pattern)
	assert.Equal(t, "rspec", settings.Config.Command)
	assert.NoError(t, settings.Validate())

	settings.Pattern = nil
	assert.EqualError(t, settings.Validate(), "no test files or directories specified")
}

func TestRenderTemplateReturnsParseError(t *testing.T) {
	_, err := RenderTemplate("{{", nil)
	assert.Error(t, err)
}

func TestCollectExamplesReturnsMissingFileError(t *testing.T) {
	_, err := (&Config{PersistenceFile: filepath.Join(t.TempDir(), "missing.txt")}).CollectExamples()
	assert.Error(t, err)
}

func TestCollectExamplesRejectsMalformedAndOversizedRows(t *testing.T) {
	tests := []struct {
		name    string
		row     string
		wantErr string
	}{
		{name: "malformed", row: "not a persistence row", wantErr: "malformed rspec persistence row"},
		{name: "oversized", row: strings.Repeat("x", 70*1024), wantErr: "token too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "examples.txt")
			contents := "example_id | status | run_time |\n-----------|--------|----------|\n" + tt.row + "\n"
			require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

			_, err := (&Config{PersistenceFile: path}).CollectExamples()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
