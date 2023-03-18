package internal

import (
	"strings"
)

type RspecExample struct {
	Id string
	Status string
}

func (r *RspecExample) Failed() bool {
	return r.Status == "failed"
}

func (r *RspecExample) Passed() bool {
	return r.Status == "passed"
}

func (r *RspecExample) Filename() string {
	return strings.Split(r.Id, "[")[0]
}

func ParseRspecExample(line string) RspecExample {
	parts := strings.Split(line, "|")

	return RspecExample{
		Id: strings.TrimSpace(parts[0]),
		Status: strings.TrimSpace(parts[1]),
	}
}

func FindFlakies(firstRun []RspecExample, secondRun []RspecExample) []RspecExample {
	var flakies []RspecExample

	for _, example := range firstRun {
		if example.Failed() {
			for _, second_example := range secondRun {
				if example.Id == second_example.Id && second_example.Passed() {
					flakies = append(flakies, example)
				}
			}
		}
	}

	return flakies
}
