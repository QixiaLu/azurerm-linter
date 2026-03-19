package azbp015

import (
	acceptance "testdata/src/mockacceptance"
	check "testdata/src/mockcheck"
)

// Invalid: HasValue is unnecessary when ImportStep is present in the same function
func invalidWithImportStep() {
	data := acceptance.BuildTestData(nil, "azurerm_example", "test")
	_ = check.That("azurerm_foo.bar").Key("shape").HasValue("Exadata.X11M")              // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("database_server_type").HasValue("X11M")       // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("storage_server_type").HasValue("X11M-HC")     // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("sku_name").HasValue("PremiumP1")              // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("name").HasValue("acctest-foo")                // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("location").HasValue("westus2")                // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("soap_pass_through").HasValue("false")         // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("subscription_required").HasValue("true")      // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("tags.label").HasValue("test")                 // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("tags.%").HasValue("1")                        // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("protocols.#").HasValue("1")                   // want `AZBP015`
	_ = check.That("azurerm_foo.bar").Key("recommendations.0.category").HasValue("Cost") // want `AZBP015`
	_ = data.ImportStep()
}

// Valid: other check methods are fine even with ImportStep
func validOtherChecksWithImportStep() {
	data := acceptance.BuildTestData(nil, "azurerm_example", "test")
	_ = check.That("azurerm_foo.bar").ExistsInAzure(nil)
	_ = check.That("azurerm_foo.bar").Key("id").Exists()
	_ = check.That("azurerm_foo.bar").Key("name").IsNotEmpty()
	_ = check.That("azurerm_foo.bar").Key("name").IsEmpty()
	_ = check.That("azurerm_foo.bar").Key("name").IsSet()
	_ = check.That("azurerm_foo.bar").Key("name").DoesNotExist()
	_ = check.That("azurerm_foo.bar").Key("id").MatchesRegex("^/subscriptions/")
	_ = data.ImportStep()
}

// Valid: HasValue without ImportStep in the function is fine (e.g. data source tests)
func validHasValueNoImportStep() {
	_ = check.That("data.azurerm_foo.bar").Key("example_property").HasValue("bar")
	_ = check.That("data.azurerm_foo.bar").Key("location").HasValue("westus2")
	_ = check.That("data.azurerm_foo.bar").Key("sku_name").HasValue("Standard")
}
