package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNullReporterAcceptsAllOperations(t *testing.T) {
	reporter := &NullReporter{}
	assert.NoError(t, reporter.Init())
	assert.NoError(t, reporter.Verify())
	assert.NoError(t, reporter.ReportFlaky([]RspecExample{{Id: "spec/flaky_spec.rb[1:1]"}}))
}
