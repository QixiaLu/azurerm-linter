package azsd003

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"field_with_exactlyone": {
			Type:         schema.TypeString,
			Optional:     true,
			ExactlyOneOf: []string{"field_with_exactlyone", "field_alternative"},
		},

		"field_with_conflicts": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"another_field"},
		},

		"pipeline": {
			Type:          schema.TypeList,
			Optional:      true,
			ExactlyOneOf:  []string{"pipeline", "pipeline_name"},
			ConflictsWith: []string{"pipeline_parameters"}, // Different field - OK
		},
	}
}

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"field_both": { // want `AZSD003`
			Type:          schema.TypeString,
			Optional:      true,
			ExactlyOneOf:  []string{"field_both", "field_other"},
			ConflictsWith: []string{"field_other"}, // Redundant - field_other is in ExactlyOneOf
		},

		"partial_overlap": { // want `AZSD003`
			Type:          schema.TypeString,
			Optional:      true,
			ExactlyOneOf:  []string{"partial_overlap", "field_x"},
			ConflictsWith: []string{"field_x", "field_y"}, // field_x overlaps
		},
	}
}
