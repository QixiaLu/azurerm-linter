package aznr007

func validNoBlockContext() {
	_ = `  name = "acckv%[1]d"`
}

func invalidMultilineRaw() {
	_ = `
resource "azurerm_resource_group" "test" {
  name     = "badname%d"` // want `AZNR007`
}

func invalidMultilineRawDeep() string {
	return `
resource "azurerm_resource_group" "test" {
  location = "%s"
  name     = "mybadresource%d"` // want `AZNR007`
}

func invalidResourceAcckv() {
	_ = `
resource "azurerm_key_vault" "test" {
  name = "acckv%[1]d"` // want `AZNR007`
}

func invalidResourceAligned() {
	_ = `
resource "azurerm_app_service_plan" "test" {
  name                = "badname%d"` // want `AZNR007`
}

func invalidResourceArbitrary() {
	_ = `
resource "azurerm_storage_account" "test" {
  name = "sdsds"` // want `AZNR007`
}

func validMultilineRaw() {
	_ = `
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}
`
}

func validDataBlock() {
	_ = `
data "azurerm_dns_zone" "test" {
  name                = "example.com"
  resource_group_name = "rg-test"
}
`
}

func validDataBlockNoAcctest() {
	_ = `
data "azurerm_resource_group" "test" {
  name = "myexistingrg"
}
`
}

func validDataBlockAfterResource() {
	_ = `
resource "azurerm_search_service" "test" {
  name                = "acctestsearchservice%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "standard"
}

data "azurerm_search_service" "test" {
  name                = "%ssds"
  resource_group_name = azurerm_resource_group.test.name
}
`
}

func validPrivateDnsZone() {
	_ = `
resource "azurerm_private_dns_zone" "test" {
  name                = "privatelink.azuredatabricks.net"
  resource_group_name = azurerm_resource_group.test.name
}
`
}

func validPrivateDnsZoneNonAcctest() {
	_ = `
resource "azurerm_private_dns_zone" "test" {
  name                = "myzone"
  resource_group_name = azurerm_resource_group.test.name
}
`
}

func validInterpolatedName() {
	_ = `
resource "azurerm_storage_account" "test" {
  name = "${azurerm_resource_group.test.name}-replica"
}
`
}

func validExcludedConfigurationName() {
	_ = `
resource "azurerm_postgresql_flexible_server_configuration" "test" {
  name  = "log_checkpoints"
  value = "on"
}
`
}
