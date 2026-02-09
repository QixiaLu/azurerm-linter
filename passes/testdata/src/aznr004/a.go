package aznr004

// Test types
type NetworkACLs struct {
	Name string
}

type NetworkRuleSet struct {
	Rules []string
}

func flattenReturnsEmptySlice(input *NetworkRuleSet) []NetworkACLs {
	if input == nil {
		return []NetworkACLs{}
	}
	return []NetworkACLs{{Name: "test"}}
}

func expandCanReturnNil(input []NetworkACLs) *NetworkRuleSet {
	if len(input) == 0 {
		return nil
	}
	return &NetworkRuleSet{}
}

func flattenToString(input *NetworkRuleSet) string {
	if input == nil {
		return ""
	}
	return "result"
}

func flattenMultipleSlicesValid(input *NetworkRuleSet) ([]NetworkACLs, []interface{}, error) {
	if input == nil {
		return []NetworkACLs{}, []interface{}{}, nil
	}
	return []NetworkACLs{{Name: "test"}}, []interface{}{"a"}, nil
}

func flattenReturnsNil(input *NetworkRuleSet) []NetworkACLs {
	if input == nil {
		return nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}
}

func flattenWithErrorNil(input *NetworkRuleSet) ([]NetworkACLs, error) {
	if input == nil {
		return nil, nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}, nil
}

func flattenMultipleSlicesOneNil(input *NetworkRuleSet) ([]NetworkACLs, []interface{}, error) {
	if input == nil {
		return []NetworkACLs{}, nil, nil // want `AZNR004`
	}
	return []NetworkACLs{{Name: "test"}}, []interface{}{"a"}, nil
}
