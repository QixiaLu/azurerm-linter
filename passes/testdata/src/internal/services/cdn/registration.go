package cdn

import (
	"testdata/src/mockpkg/pluginsdk"
	"testdata/src/mockpkg/sdk"
)

type Registration struct{}

func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{ // want `AZNR005`
		"azurerm_managed_disk":     nil,
		"azurerm_availability_set": nil,
	}
}

func (r Registration) Resources() []sdk.Resource {
	return nil
}
