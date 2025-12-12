package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Test: Resource without SetID call - should be skipped
func resourceNoSetID() *schema.Resource {
	return &schema.Resource{
		Create: noSetIDResourceCreate,

		Schema: map[string]*schema.Schema{
			// Without ID extraction, just alphabetical by category
			"location": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func noSetIDResourceCreate(d *schema.ResourceData, meta interface{}) error {
	// No SetID call
	return nil
}
