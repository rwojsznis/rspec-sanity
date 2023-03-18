package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRspecExample(t *testing.T) {
	example := ParseRspecExample("./spec/flaky_spec.rb[1:1]        | passed | 0.00029 seconds |")

	assert.Equal(t, example.Id, "./spec/flaky_spec.rb[1:1]")
	assert.Equal(t, example.Status, "passed")
}

func TestFindFlakies(t *testing.T) {
	firstRun := []RspecExample{
		{Id: "./spec/some_other_spec.rb[1:1]", Status: "failed"},
		{Id: "./spec/some_spec.rb[2:2]", Status: "failed"},
		{Id: "./spec/test_spec.rb[1:3]", Status: "passed"},
		{Id: "./spec/yet_another.rb[1:100]", Status: "failed"},
	}

	secondRun := []RspecExample{
		{Id: "./spec/some_other_spec.rb[1:1]", Status: "failed"},
		{Id: "./spec/some_spec.rb[2:2]", Status: "passed"},
		{Id: "./spec/test_spec.rb[1:3]", Status: "passed"},
		{Id: "./spec/yet_another.rb[1:100]", Status: "passed"},
	}

	expected := []RspecExample{
		{Id: "./spec/some_spec.rb[2:2]", Status: "failed"},
		{Id: "./spec/yet_another.rb[1:100]", Status: "failed"},
	}

	assert.Equal(t, FindFlakies(firstRun, secondRun), expected)
	assert.Empty(t, FindFlakies(firstRun, firstRun))
	assert.Empty(t, FindFlakies(secondRun, secondRun))
}

func TestRspecExampleFilename(t *testing.T) {
	example := RspecExample{Id: "./spec/some_spec.rb[2:2]"}
	assert.Equal(t, example.Filename(), "./spec/some_spec.rb")
}
