package aznr008

// Invalid: hardcoded resource IDs with subscription GUIDs in HCL config
func invalidHardcodedId() {
	_ = `  source_autonomous_database_id = "/subscriptions/049e5678-fbb1-4861-93f3-7528bd0779fd/resourceGroups/19CTest/providers/Oracle.Database/autonomousDatabases/adbsTestPublicClone"` // want `AZNR008`
}

func invalidDummyGuid() {
	_ = `  msi_work_space_resource_id = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/test/providers/Microsoft.Databricks/workspaces/testworkspace"` // want `AZNR008`
}

func invalidZeroGuid() {
	_ = `  webhook_resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg-runbooks/providers/microsoft.automation/automationaccounts/aaa001/webhooks/webhook_alert"` // want `AZNR008`
}

func invalidMultilineConfig() {
	_ = `  some_id = "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resGroup1/providers/Microsoft.Network/virtualNetworks/vnet1"` // want `AZNR008`
}

// Valid: no hardcoded IDs - uses interpolation
func validInterpolation() {
	_ = `
  source_id = azurerm_resource.test.id
`
}

// Valid: uses fmt.Sprintf placeholder
func validFmtSprintf() {
	_ = `
  source_id = %[2]s
`
}

// Valid: not a resource ID pattern
func validNotResourceId() {
	_ = `
  description = "This references /subscriptions/ in documentation text"
`
}

// Valid: partial path without full GUID
func validPartialPath() {
	_ = `
  path = "/subscriptions/"
`
}

func invalidMultilineRaw() {
	_ = `
resource "azurerm_example" "test" {
  some_id = "/subscriptions/049e5678-fbb1-4861-93f3-7528bd0779fd/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet1"` // want `AZNR008`
}

func invalidMultilineRawDeep() {
	_ = `
resource "azurerm_example" "test" {
  name     = "acctestexample"
  location = "westeurope"
  target   = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1"` // want `AZNR008`
}
