package azrn002

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"enabled": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"disk_is_attached": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"is_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"is_enabled": { // want `AZRN002`
			Type:     schema.TypeBool,
			Optional: true,
		},
	}
}
