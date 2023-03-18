package internal

import (
	"fmt"
	"os"
)

type JiraConfig struct {
	EpicId string `toml:"epic_id,omitempty"`
	ProjectId string `toml:"project_id,omitempty"`
	TaskTypeId string `toml:"task_type_id,omitempty"`
	Template string `toml:"template,omitempty"`
	Labels []string `toml:"labels,omitempty"`
	token string
	user string
	host string
}

func (jc *JiraConfig) GetUser() string {
	return jc.user
}

func (jc *JiraConfig) GetToken() string {
	return jc.token
}

func (jc *JiraConfig) GetHost() string {
	return jc.host
}

func (jc *JiraConfig) Prepare() error {
	if jc.EpicId == "" {
		return fmt.Errorf("no jira epic id specified in config")
	}

	if jc.ProjectId == "" {
		return fmt.Errorf("no jira project id specified in config")
	}

	if jc.TaskTypeId == "" {
		return fmt.Errorf("no jira task type id specified in config")
	}

	if jc.Template == "" {
		return fmt.Errorf("no jira template specified in config")
	}

	token, present := os.LookupEnv("RSPEC_SANITY_JIRA_TOKEN")
	if !present {
		return fmt.Errorf("specify jira token under RSPEC_SANITY_JIRA_TOKEN env")
	}
	jc.token = token

	user, present := os.LookupEnv("RSPEC_SANITY_JIRA_USER")
	if !present {
		return fmt.Errorf("specify jira user under RSPEC_SANITY_JIRA_USER env")
	}
	jc.user = user

	host, present := os.LookupEnv("RSPEC_SANITY_JIRA_HOST")
	if !present {
		return fmt.Errorf("specify jira full host (including scheme) under RSPEC_SANITY_JIRA_HOST env")
	}
	jc.host = host


	return nil
}
