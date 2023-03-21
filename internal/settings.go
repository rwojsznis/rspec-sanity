package internal

import (
	// "fmt"

	"fmt"

	"github.com/urfave/cli/v2"
)

type Settings struct {
	SkipRerun  bool
	ConfigPath string
	Config     Config
	Pattern    []string
}

func (s *Settings) Load(cCtx *cli.Context) error {
	pattern := cCtx.Args().Slice()
	s.Pattern = pattern

	config, err := LoadConfig(s.ConfigPath)

	if err != nil {
		return err
	}

	s.Config = *config
	return nil
}

func (s *Settings) Validate() error {
	if len(s.Pattern) == 0 {
		return fmt.Errorf("no test files or directories specified")
	}

	return nil
}
