package azsd005

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var schemaBuckets = []map[string]*schema.Schema{
	invalidManualListingWithNone(),
	invalidManualListingViaVariable(),
	invalidPossibleValuesForWithNone(),
	validManualListingExcludingNone(),
	validPossibleValuesForWithoutNone(),
	validLiteralStringNone(),
}

// Invalid: manual listing includes string(virtualmachines.ShutdownOnIdleModeNone)
func invalidManualListingWithNone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"shutdown_on_idle": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{ // want `AZSD005`
				string(virtualmachines.ShutdownOnIdleModeNone),
				string(virtualmachines.ShutdownOnIdleModeUserAbsence),
				string(virtualmachines.ShutdownOnIdleModeLowUsage),
			}, false),
		},
	}
}

// Invalid: manual listing via variable includes None
func invalidManualListingViaVariable() map[string]*schema.Schema {
	values := []string{
		string(virtualmachines.ShutdownOnIdleModeNone),
		string(virtualmachines.ShutdownOnIdleModeUserAbsence),
		string(virtualmachines.ShutdownOnIdleModeLowUsage),
	}

	return map[string]*schema.Schema{
		"shutdown_on_idle": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice(values, false), // want `AZSD005`
		},
	}
}

// Invalid: PossibleValuesFor* function where enum has a "None" constant
func invalidPossibleValuesForWithNone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"shutdown_on_idle": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice(virtualmachines.PossibleValuesForShutdownOnIdleMode(), false), // want `AZSD005`
		},
	}
}

// Valid: manual listing that correctly excludes None
func validManualListingExcludingNone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"shutdown_on_idle": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(virtualmachines.ShutdownOnIdleModeUserAbsence),
				string(virtualmachines.ShutdownOnIdleModeLowUsage),
			}, false),
		},
	}
}

// Valid: PossibleValuesFor* function where enum has no "None" constant
func validPossibleValuesForWithoutNone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"priority": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice(virtualmachines.PossibleValuesForVirtualMachinePriorityTypes(), false),
		},
	}
}

// Valid: literal string "None" is not from an SDK enum type
func validLiteralStringNone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mode": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				"None",
				"Active",
				"Passive",
			}, false),
		},
	}
}
