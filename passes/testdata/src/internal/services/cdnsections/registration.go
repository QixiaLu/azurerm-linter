package cdnsections

import "testdata/src/mockpkg/pluginsdk"

type Registration struct{}

func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
	resources := map[string]*pluginsdk.Resource{ // want `AZNR005`
		// Compute
		"azurerm_managed_disk":     nil,
		"azurerm_availability_set": nil,

		// VM
		"azurerm_virtual_machine": nil,
		"azurerm_dedicated_host":  nil,
	}

	return resources
}

func (r Registration) GloballyUnsortedAcrossSections() map[string]*pluginsdk.Resource {
	resources := map[string]*pluginsdk.Resource{ // want `AZNR005`
		// VM
		"azurerm_dedicated_host":  nil,
		"azurerm_virtual_machine": nil,

		// Compute
		"azurerm_availability_set": nil,
		"azurerm_managed_disk":     nil,
	}

	return resources
}
