package internal

import (
	"fmt"
	"os"
)

type GithubConfig struct {
	Owner string `toml:"owner,omitempty"`
	Repo string `toml:"repo,omitempty"`
	Template string `toml:"template,omitempty"`
	Labels []string `toml:"labels,omitempty"`
	Reopen bool `toml:"reopen,omitempty"`
	token string
}

func (gc *GithubConfig) Prepare() error {
	if gc.Owner == "" {
		return fmt.Errorf("no github owner specified in config")
	}

	if gc.Repo == "" {
		return fmt.Errorf("no github repo specified in config")
	}

	if gc.Template == "" {
		return fmt.Errorf("no github template specified in config")
	}

	token, present := os.LookupEnv("RSPEC_SANITY_GITHUB_TOKEN")

	if !present {
		return fmt.Errorf("specify github token under RSPEC_SANITY_GITHUB_TOKEN env")
	}

	gc.token = token

	return nil
}
func (gc *GithubConfig) GetToken() string {
	return gc.token
}
