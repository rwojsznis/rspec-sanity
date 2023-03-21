package main

import (
	"log"
	"os"

	internal "github.com/rwojsznis/rspec-sanity/internal"
	"github.com/urfave/cli/v2"
)

func main() {
	settings := internal.Settings{}

	app := &cli.App{
		Usage:     "a tool that helps you to ticket flaky tests in your RSpec suite",
		ArgsUsage: "[test files or directories]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "skip-rerun",
				Usage:       "Do not re-run the tests (also skips reporting)",
				Destination: &settings.SkipRerun,
			},
			&cli.StringFlag{
				Name:        "config",
				DefaultText: ".rspec-sanity.toml",
				Usage:       "Load configuration from `FILE`",
				Destination: &settings.ConfigPath,
				Value:       ".rspec-sanity.toml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "verify",
				Usage: "verify configuration - will try to add a test issue report to Jira/Github",
				Action: func(cCtx *cli.Context) error {
					err := settings.Load(cCtx)
					if err != nil {
						return err
					}

					reporter := settings.Config.GetReporter()
					err = reporter.Init()
					if err != nil {
						return err
					}
					
					return reporter.Verify()
				},
			},
			{
				Name:  "run",
				Usage: "run rspec according to the configuration",
				Action: func(cCtx *cli.Context) error {
					err := settings.Load(cCtx)
					if err != nil {
						return err
					}

					err = settings.Validate()
					if err != nil {
						return err
					}

					runner := internal.Runner{
						Settings: &settings,
					}

					runnerStatus := runner.Run()

					if runnerStatus.HasFlakies() {
						// we will crash app on error here; otherwise debugging potential
						// issues in reporter itself will be nightmare
						reporter := settings.Config.GetReporter()
						err = reporter.Init()

						if err != nil {
							return err
						}

						err = internal.ReportFlakies(reporter, runnerStatus.FlakyExamples)

						if err != nil {
							return err
						}

					} else {
						log.Println("[rspec-sanity] No flaky examples found")
					}

					// if nothing failed during reporting - propagate exit code from rspec
					os.Exit(runnerStatus.StatusCode)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
