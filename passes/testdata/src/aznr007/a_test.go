package aznr007

// Valid: name starts with "acctest"
func validAcctest() {
	_ = `  name = "acctestkv%[1]d"`
}

// Valid: name starts with "acctestRG"
func validAcctestRG() {
	_ = `  name = "acctestRG-df-%d"`
}

// Valid: name with extra spaces before = (alignment)
func validAcctestAligned() {
	_ = `  name                = "acctestresource%d"`
}

// Valid: not the "name" field - display_name should not be checked
func validDisplayName() {
	_ = `  display_name = "somename"`
}

// Valid: name_override is not the "name" field
func validNameOverride() {
	_ = `  name_override = "somebar"`
}

// Valid: name uses variable reference (no string value)
func validVarRef() {
	_ = `  name = var.name`
}

// Valid: name is empty string (regex requires at least one char)
func validEmpty() {
	_ = `  name = ""`
}

// Valid: nested name (4-space indent) - not a top-level resource name
func validNestedName() {
	_ = `    name = "ascslb"`
}

// Valid: deeply nested name (6-space indent)
func validDeeplyNested() {
	_ = `      name = "dblb"`
}

// Valid: nested name inside a block
func validNestedBlock() {
	_ = `    name  = "First"`
}

// Invalid: top-level name starts with "acc" but not "acctest"
func invalidAcckv() {
	_ = `  name = "acckv%[1]d"` // want `AZNR007`
}

// Invalid: top-level name starts with "acc" but not "acctest"
func invalidAccsa() {
	_ = `  name = "accsa%[4]s"` // want `AZNR007`
}

// Invalid: top-level name doesn't start with "acctest"
func invalidCredential() {
	_ = `  name = "credential%d"` // want `AZNR007`
}

// Invalid: top-level name doesn't start with "acctest"
func invalidTestresource() {
	_ = `  name = "testresource%d"` // want `AZNR007`
}

// Invalid: arbitrary top-level name not starting with "acctest"
func invalidArbitrary() {
	_ = `  name = "sdsds"` // want `AZNR007`
}

// Invalid: top-level name with alignment spaces before =
func invalidAligned() {
	_ = `  name                = "badname%d"` // want `AZNR007`
}

// Invalid: multiple top-level violations in one string
func invalidMultiple() {
	_ = "  name = \"acckv1%d\"\n  name = \"myresource%d\"" // want `AZNR007` `AZNR007`
}
