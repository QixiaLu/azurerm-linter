package aznr004

// Test types
type NetworkACLs struct {
	Name string
}

type NetworkRuleSet struct {
	Rules []string
}

// Valid: Returns empty slice literal
func flattenReturnsEmptySlice(input *NetworkRuleSet) []NetworkACLs {
	if input == nil {
		return []NetworkACLs{}
	}
	return []NetworkACLs{{Name: "test"}}
}

// Valid: Not a flatten function (expand can return nil)
func expandCanReturnNil(input []NetworkACLs) *NetworkRuleSet {
	if len(input) == 0 {
		return nil
	}
	return &NetworkRuleSet{}
}

// Valid: Returns non-slice type (string can be empty)
func flattenToString(input *NetworkRuleSet) string {
	if input == nil {
		return ""
	}
	return "result"
}

// Valid: Multiple slice returns, all return empty slices
func flattenMultipleSlicesValid(input *NetworkRuleSet) ([]NetworkACLs, []interface{}, error) {
	if input == nil {
		return []NetworkACLs{}, []interface{}{}, nil
	}
	return []NetworkACLs{{Name: "test"}}, []interface{}{"a"}, nil
}

// Invalid: Returns nil instead of empty slice
func flattenReturnsNil(input *NetworkRuleSet) []NetworkACLs {
	if input == nil {
		return nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}
}

// Invalid: Multiple return values with nil slice
func flattenWithErrorNil(input *NetworkRuleSet) ([]NetworkACLs, error) {
	if input == nil {
		return nil, nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}, nil
}

// Invalid: Multiple slice returns, one returns nil
func flattenMultipleSlicesOneNil(input *NetworkRuleSet) ([]NetworkACLs, []interface{}, error) {
	if input == nil {
		return []NetworkACLs{}, nil, nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}, []interface{}{"a"}, nil
}
