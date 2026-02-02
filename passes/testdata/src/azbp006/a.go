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

// Invalid: redundant nil assignments to pointer fields
func invalidCases() *Config {
	return &Config{
		Name:    nil, // want `AZBP006`
		Options: nil, // want `AZBP006`
	}
}

// Valid: slice and map nil assignments are OK (not pointers)
func validSliceAndMap() *Config {
	return &Config{
		Items: nil,
		Data:  nil,
	}
}

// Valid: no nil assignments
func validOmitted() *Config {
	return &Config{}
}
