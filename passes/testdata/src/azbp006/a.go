package azbp006

type Config struct {
	Name    *string
	Options *Options
	Items   []string          // slice - should NOT be flagged
	Data    map[string]string // map - should NOT be flagged
}

type Options struct {
	Enabled bool
}

func invalidCases() *Config {
	return &Config{
		Name:    nil, // want `AZBP006`
		Options: nil, // want `AZBP006`
	}
}

func validSliceAndMap() *Config {
	return &Config{
		Items: nil,
		Data:  nil,
	}
}

func validOmitted() *Config {
	return &Config{}
}
