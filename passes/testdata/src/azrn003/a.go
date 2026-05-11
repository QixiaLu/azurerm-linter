package azrn003

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vmss_zonal_upgrade_mode": { // want `AZRN003`
			Type:     schema.TypeString,
			Optional: true,
		},
		"min_duration": { // want `AZRN003`
			Type:     schema.TypeString,
			Optional: true,
		},
		"max_duration": { // want `AZRN003`
			Type:     schema.TypeString,
			Optional: true,
		},
		"vm_name": { // want `AZRN003`
			Type:     schema.TypeString,
			Optional: true,
		},
		"size_gb": { // want `AZRN003`
			Type:     schema.TypeInt,
			Optional: true,
		},
	}
}
