package azbp008

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Invalid: using string(enum) in StringInSlice instead of PossibleValuesFor
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

// Valid: using PossibleValuesFor function
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

// Valid: using literal strings (not SDK enums)
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

// Valid: mixed enum and literal string (should not trigger)
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

// Valid: mixed enum and literal string (should not trigger)
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
