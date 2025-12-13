package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Valid: Optional+Computed with correct order and NOTE: O+C comment
		"field1": {
			Type:     schema.TypeString,
			Optional: true,
			// NOTE: O+C - field can be set by user or computed from API when not provided
			Computed: true,
		},

		// Valid: Optional+Computed with correct order and NOTE: O+C comment (different style)
		"field2": {
			Type:     schema.TypeString,
			Optional: true,
			// NOTE: O+C - defaults to value from parent resource if not specified
			Computed: true,
		},

		// Valid: Only Optional (no Computed, so no rule violation)
		"field3": {
			Type:     schema.TypeString,
			Optional: true,
		},

		// Valid: Only Computed (no Optional, so no rule violation)
		"field4": {
			Type:     schema.TypeString,
			Computed: true,
		},

		// Valid: Required field (not Optional+Computed)
		"field5": {
			Type:     schema.TypeString,
			Required: true,
		},
	}
}

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Invalid: Optional+Computed in wrong order (Computed before Optional)
		"wrong_order": { // want `AZC003: field "wrong_order" has Optional and Computed in wrong order \(Optional must come before Computed\)`
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},

		// Invalid: Optional+Computed with correct order but missing NOTE: O+C comment
		"missing_comment": { // want `AZC003: field "missing_comment" is Optional\+Computed but missing required comment. Add '// NOTE: O\+C - <explanation>' between Optional and Computed`
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},

		// Invalid: Optional+Computed with comment but wrong format (no "NOTE: O+C")
		"wrong_comment_format": { // want `AZC003: field "wrong_comment_format" is Optional\+Computed but missing required comment. Add '// NOTE: O\+C - <explanation>' between Optional and Computed`
			Type:     schema.TypeString,
			Optional: true,
			// This field can be computed
			Computed: true,
		},
	}
}
