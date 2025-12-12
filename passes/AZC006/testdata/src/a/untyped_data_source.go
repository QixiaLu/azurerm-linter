package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Test: Data source with correct order
func unTypedDataSourceValid() *schema.Resource {
	return &schema.Resource{
		Read: unTypedDataSourceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"account_tier": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

// Test: Data source with wrong order
func unTypedDataSourceInvalid() *schema.Resource {
	return &schema.Resource{
		Read: unTypedDataSourceRead,

		Schema: map[string]*schema.Schema{ // want `name, resource_group_name, location, account_tier, tags`
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"account_tier": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func unTypedDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	resourceGroupName := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	subscriptionId := "sub"

	id := parse.NewResourceID(subscriptionId, resourceGroupName, name)
	d.SetId(id.ID())
	return nil
}
