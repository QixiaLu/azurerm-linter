package azbp007

type Config struct {
	Name  string
	Names *[]string
}

func invalidCases() {
	_ = []string{}     // want `AZBP007`
	var x = []string{} // want `AZBP007`
	_ = x
}

func validMake() {
	_ = make([]string, 0)
	var x = make([]string, 0)
	_ = x
}

func validOtherSlices() {
	_ = []int{}
	_ = []Config{}
	_ = []interface{}{}
}

func validNonEmpty() {
	_ = []string{"a", "b"}
}

func validInlineUsage() {
	_ = Config{Names: &[]string{}}
	_ = &[]string{}
	takesSlice([]string{})
}

func takesSlice(_ []string) {}

func validTestTablePattern() {
	tests := []struct {
		name     string
		from     []string
		to       []string
		expected []string
	}{
		{
			name:     "case 1",
			from:     []string{"a", "b", "c"},
			to:       []string{"a", "c"},
			expected: []string{"b"},
		},
		{
			name:     "empty expected",
			from:     []string{"a", "b", "c"},
			to:       []string{"a", "b", "c"},
			expected: []string{},
		},
	}
	_ = tests
}
