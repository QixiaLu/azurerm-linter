package azsd004

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func attributes() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"a": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{ // want `AZSD004`
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
		"b": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"c": { // want `AZSD004`
			Type:         schema.TypeString,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		"d": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"property1": { // want `AZSD004`
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},
					"property2": {
						Type:     schema.TypeInt,
						Computed: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"property1": { // want `AZSD004`
									Type:     schema.TypeString,
									Required: true,
								},
								"property2": {
									Type:     schema.TypeInt,
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}
}
