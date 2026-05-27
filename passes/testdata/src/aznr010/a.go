package aznr010

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func validCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}
}

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The name of the resource.", // want `AZNR010`
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}
}
