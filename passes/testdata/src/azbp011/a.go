package azbp011

import (
	"strings"

	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
)

func badStringCastingComparison() bool {
	result1 := strings.EqualFold(string(virtualmachines.VirtualMachinePriorityTypesSpot), string(virtualmachines.VirtualMachinePriorityTypesRegular)) // want `AZBP011`
	result2 := strings.EqualFold(string(virtualmachines.VirtualMachinePriorityTypesLow), string(virtualmachines.VirtualMachinePriorityTypesSpot))     // want `AZBP011`

	return result1 || result2
}

func legitimateStringEqualFold() bool {
	userInput := "Low"

	result1 := strings.EqualFold(userInput, string(virtualmachines.VirtualMachinePriorityTypesLow))
	return result1
}
