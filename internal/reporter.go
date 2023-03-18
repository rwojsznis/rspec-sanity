package internal

type Reporter interface {
	Init() error
	ReportFlaky([]RspecExample) error
}

func ReportFlakies(reporter Reporter, flakies []RspecExample) error {
	groups := make(map[string][]RspecExample)
	for _, example := range flakies {
		groups[example.Filename()] = append(groups[example.Filename()], example)
	}

	for _, group := range groups {
		err := reporter.ReportFlaky(group)
		if err != nil {
			return err
		}
	}

	return nil
}
