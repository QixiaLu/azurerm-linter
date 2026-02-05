package aznr005

import (
	"testdata/src/mockpkg/pluginsdk"
	"testdata/src/mockpkg/sdk"
)

type Registration struct{}

func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{
		"azurerm_availability_set":    nil,
		"azurerm_dedicated_host":      nil,
		"azurerm_disk_encryption_set": nil,
		"azurerm_managed_disk":        nil,
		"azurerm_virtual_machine":     nil,
	}
}

func (r Registration) SupportedDataSources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{}
}

func (r Registration) InvalidSupportedResources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{ // want `AZNR005`
		"azurerm_availability_set":       nil,
		"azurerm_dedicated_host":         nil,
		"azurerm_managed_disk":           nil,
		"azurerm_disk_encryption_set":    nil,
		"azurerm_ssh_public_key":         nil,
		"azurerm_managed_disk_sas_token": nil,
	}
}

func (r Registration) Resources() []sdk.Resource {
	return []sdk.Resource{
		ApiManagementNotificationRecipientEmailResource{},
		ApiManagementNotificationRecipientUserResource{},
	}
}

func (r Registration) InvalidResources() []sdk.Resource {
	return []sdk.Resource{ // want `AZNR005`
		ApiManagementNotificationRecipientUserResource{},
		ApiManagementNotificationRecipientEmailResource{},
	}
}

type ApiManagementNotificationRecipientEmailResource struct{}
type ApiManagementNotificationRecipientUserResource struct{}

func (ApiManagementNotificationRecipientEmailResource) Arguments() map[string]*pluginsdk.Schema {
	return nil
}
func (ApiManagementNotificationRecipientEmailResource) Attributes() map[string]*pluginsdk.Schema {
	return nil
}
func (ApiManagementNotificationRecipientEmailResource) ModelObject() interface{} { return nil }
func (ApiManagementNotificationRecipientEmailResource) ResourceType() string     { return "mock" }
func (ApiManagementNotificationRecipientEmailResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
func (ApiManagementNotificationRecipientEmailResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
func (ApiManagementNotificationRecipientEmailResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

func (ApiManagementNotificationRecipientUserResource) Arguments() map[string]*pluginsdk.Schema {
	return nil
}
func (ApiManagementNotificationRecipientUserResource) Attributes() map[string]*pluginsdk.Schema {
	return nil
}
func (ApiManagementNotificationRecipientUserResource) ModelObject() interface{} { return nil }
func (ApiManagementNotificationRecipientUserResource) ResourceType() string     { return "mock" }
func (ApiManagementNotificationRecipientUserResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
func (ApiManagementNotificationRecipientUserResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
func (ApiManagementNotificationRecipientUserResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
