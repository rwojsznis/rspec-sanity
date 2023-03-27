package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockReporter struct {
	Groups map[string][]RspecExample
}

func (m *MockReporter) Init() error {
	m.Groups = make(map[string][]RspecExample)
	return nil
}

func (m *MockReporter) ReportFlaky(flakies []RspecExample) error {
	filename := flakies[0].Filename()
	m.Groups[filename] = append(m.Groups[filename], flakies...)
	return nil
}

func (m *MockReporter) Verify() error {
	return nil
}

func TestReportFlakies(t *testing.T) {
	reporter := &MockReporter{}
	err := reporter.Init()
	assert.NoError(t, err)

	err = ReportFlakies(reporter, []RspecExample{
		{Id: "./spec/flaky_spec.rb[1:1]"},
		{Id: "./spec/flaky_spec.rb[1:2]"},
		{Id: "./spec/flaky_spec.rb[1:3]"},
		{Id: "./spec/new_flaky_spec.rb[1:1]"},
	})
 
	assert.NoError(t, err)

	assert.Equal(t, 2, len(reporter.Groups))

	assert.Equal(t, 3, len(reporter.Groups["./spec/flaky_spec.rb"]))
	assert.Equal(t, "./spec/flaky_spec.rb[1:2]", reporter.Groups["./spec/flaky_spec.rb"][1].Id)

	assert.Equal(t, 1, len(reporter.Groups["./spec/new_flaky_spec.rb"]))
	assert.Equal(t, "./spec/new_flaky_spec.rb[1:1]", reporter.Groups["./spec/new_flaky_spec.rb"][0].Id)
}
