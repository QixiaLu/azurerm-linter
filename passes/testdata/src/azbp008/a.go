package azbp008

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func invalidCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"priority": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  string(virtualmachines.VirtualMachinePriorityTypesRegular),
			ValidateFunc: validation.StringInSlice([]string{ // want `AZBP008`
				string(virtualmachines.VirtualMachinePriorityTypesLow),
				string(virtualmachines.VirtualMachinePriorityTypesRegular),
				string(virtualmachines.VirtualMachinePriorityTypesSpot),
			}, false),
		},
	}
}

func validCases() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"priority": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(virtualmachines.VirtualMachinePriorityTypesRegular),
			ValidateFunc: validation.StringInSlice(virtualmachines.PossibleValuesForVirtualMachinePriorityTypes(), false),
		},
	}
}

func validLiteralStrings() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"sku_name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.StringInSlice([]string{
				"Standard_2G",
				"Standard_4G",
				"Standard_8G",
			}, false),
		},
	}
}

func validMixedContent() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"priority": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(virtualmachines.VirtualMachinePriorityTypesLow),
				string(virtualmachines.VirtualMachinePriorityTypesRegular),
				string(virtualmachines.VirtualMachinePriorityTypesSpot),
				"custom_value",
			}, false),
		},
	}
}

func validNotSameContent() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"priority": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(virtualmachines.VirtualMachinePriorityTypesLow),
				string(virtualmachines.VirtualMachinePriorityTypesRegular),
			}, false),
		},
	}
}
