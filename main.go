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
		Usage: "a tool that helps you to ticket flaky tests in your RSpec suite",
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
		Action: func(cCtx *cli.Context) error {
			err := settings.Load(cCtx)
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

				if reporter != nil {
					err = reporter.Init()

					if err != nil {
						return err
					}

					err = internal.ReportFlakies(reporter, runnerStatus.FlakyExamples)

				if err != nil {
					return err
				}
				} else {
					log.Println("[rspec-sanity] No reporter configured, skipping!")
				}				
			} else {
				log.Println("[rspec-sanity] No flaky examples found")
			}

			// if nothing failed during reporting - propagate exit code from rspec
			os.Exit(runnerStatus.StatusCode)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
