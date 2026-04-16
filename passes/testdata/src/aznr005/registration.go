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

func (r Registration) InvalidSupportedResourcesViaVariable() map[string]*pluginsdk.Resource {
	resources := map[string]*pluginsdk.Resource{ // want `AZNR005`
		"azurerm_availability_set":       nil,
		"azurerm_dedicated_host":         nil,
		"azurerm_managed_disk":           nil,
		"azurerm_disk_encryption_set":    nil,
		"azurerm_ssh_public_key":         nil,
		"azurerm_managed_disk_sas_token": nil,
	}

	return resources
}

func (r Registration) SectionedDataSources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{ // want `AZNR005`
		// CDN
		"azurerm_cdn_profile": nil,

		// FrontDoor
		"azurerm_cdn_frontdoor_custom_domain":   nil,
		"azurerm_cdn_frontdoor_endpoint":        nil,
		"azurerm_cdn_frontdoor_firewall_policy": nil,
		"azurerm_cdn_frontdoor_origin_group":    nil,
		"azurerm_cdn_frontdoor_profile":         nil,
		"azurerm_cdn_frontdoor_rule_set":        nil,
		"azurerm_cdn_frontdoor_secret":          nil,
	}
}

func (r Registration) InvalidSectionedDataSources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{ // want `AZNR005`
		// CDN
		"azurerm_cdn_profile": nil,

		// FrontDoor
		"azurerm_cdn_frontdoor_profile":         nil,
		"azurerm_cdn_frontdoor_custom_domain":   nil,
		"azurerm_cdn_frontdoor_endpoint":        nil,
		"azurerm_cdn_frontdoor_firewall_policy": nil,
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

func (r Registration) InvalidResourcesViaVariable() []sdk.Resource {
	resources := []sdk.Resource{ // want `AZNR005`
		ApiManagementNotificationRecipientUserResource{},
		ApiManagementNotificationRecipientEmailResource{},
	}

	return resources
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
