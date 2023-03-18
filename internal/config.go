package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Command         string        `toml:"command,omitempty"`
	Arguments       string        `toml:"arguments,omitempty"`
	RerunArguments  string        `toml:"rerun_arguments,omitempty"`
	PersistenceFile string        `toml:"persistence_file,omitempty"`
	Github          *GithubConfig `toml:"github,omitempty"`
	Jira            *JiraConfig   `toml:"jira,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	_, err := os.Stat(path)

	if err != nil {
		return nil, fmt.Errorf(`error reading config file from: "%s" ("%w")`, path, err)
	}

	config := &Config{}
	_, err = toml.DecodeFile(path, &config)

	if config.Command == "" {
		return nil, fmt.Errorf("no rspec command specified in config")
	}

	if config.PersistenceFile == "" {
		return nil, fmt.Errorf(`no persistence file specified in config
Specify the path to the file where rspec stores the list of executed examples.
config.example_status_persistence_file_path = 'spec/examples.txt'
		`)
	}

	if config.Github != nil {
		err = config.Github.Prepare()
		if err != nil {
			return nil, err
		}
	}

	if config.Jira != nil {
		err = config.Jira.Prepare()
		if err != nil {
			return nil, err
		}
	}

	return config, err
}

func (c *Config) GetReporter() Reporter {
	if c.Github != nil {
		return NewGithubReporter(c.Github)
	} else if c.Jira != nil {
		return NewJiraReporter(c.Jira)
	} else {
		return nil
	}
}

func (c *Config) RunCommand(pattern []string) string {
	var cmd []string
	cmd = append(cmd, c.Command, c.Arguments)
	cmd = append(cmd, pattern...)
	cmd = removeBlanks(cmd)

	return strings.Join(cmd, " ")
}

func (c *Config) RerunCommand(pattern []string) string {
	var cmd []string
	cmd = append(cmd, c.Command, c.RerunArguments, "--only-failures")
	cmd = append(cmd, pattern...)
	cmd = removeBlanks(cmd)

	return strings.Join(cmd, " ")
}

func (c *Config) CollectExamples() ([]RspecExample, error) {
	file, err := os.Open(c.PersistenceFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var examples []RspecExample

	scanner := bufio.NewScanner(file)

	// skip headers
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		examples = append(examples, ParseRspecExample(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return examples, nil
}

type TemplateData struct {
	Examples []RspecExample
	Env      map[string]string
}

func RenderTemplate(customTemplate string, examples []RspecExample) (string, error) {
	tmpl, err := new(template.Template).Parse(customTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	env := os.Environ()
	envMap := make(map[string]string)

	for _, val := range env {
		pair := strings.SplitN(val, "=", 2)
		envMap[pair[0]] = pair[1]
	}

	data := TemplateData{
		Examples: examples,
		Env:      envMap,
	}

	err = tmpl.Execute(&buf, data)

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func removeBlanks(s []string) []string {
	var r []string
	for _, str := range s {
		trimmed := strings.TrimSpace(str)
		if trimmed != "" {
			r = append(r, trimmed)
		}
	}
	return r
}
