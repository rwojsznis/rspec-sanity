package internal

import "log"


type NullReporter struct {}

func (r *NullReporter) Init() error {
	log.Println("[null] No reporter configured, skipping init")
	return nil
}

func (r *NullReporter) Verify() error {
	log.Println("[null] No reporter configured, skipping verification")
	return nil
}

func (r *NullReporter) ReportFlaky(flakies []RspecExample) error {
	log.Printf("[null] No reporter configured, skipping flaky report: %s\n", flakies[0].Id)
	return nil
}
