package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Test: Resource with unresolvable variables - should be skipped
func resourceUnresolvableVars() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,

		Schema: map[string]*schema.Schema{
			// Unresolvable variables in ID construction
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
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

func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	// ID constructed from variables not from d.Get()
	someExternalValue := getFromSomewhere()
	resourceGroupName := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	id := parse.NewResourceID(resourceGroupName, name, someExternalValue)
	d.SetId(id.ID())
	return nil
}

func getFromSomewhere() string {
	return "external-value"
}
