package cdnazbp005

type Registration struct{}

func (r Registration) Resources() []string {
	return []string{
		"a",
		"b",
	}
}

//lintignore:AZNR005 temporary exemption
