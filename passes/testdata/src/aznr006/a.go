package aznr006

// Valid cases - no want comments
func validCases() {
	var state SomeState
	var cloneProps SomeProps

	// Good - flatten method handles nil checks internally
	state.CustomerContacts = flattenCloneCustomerContacts(cloneProps.CustomerContacts)

	// Good - direct assignment without flatten method
	if cloneProps.OtherField != nil {
		state.OtherField = *cloneProps.OtherField
	}

	// Good - not a flatten method call
	if someData != nil {
		processData(*someData)
	}

	// Good - multiple statements in if body (shouldn't be flagged)
	if cloneProps.CustomerContacts != nil {
		state.CustomerContacts = flattenCloneCustomerContacts(*cloneProps.CustomerContacts)
		state.OtherField = "processed"
	}

	// Good - nil check doesn't match the dereferenced variable
	if cloneProps.OtherField != nil {
		state.CustomerContacts = flattenCloneCustomerContacts(*cloneProps.CustomerContacts)
	}

	if cloneProps.CustomerContacts != nil {
		state.CustomerContacts = flattenPointerCustomerContacts(cloneProps.CustomerContacts)
		state.OtherField = "something"
	}
}

// Invalid cases - add want comments to each violation
func invalidCases() {
	var state SomeState
	var cloneProps SomeProps

	// Bad - nil check should be inside the flatten method
	if cloneProps.CustomerContacts != nil { // want "AZNR006:"
		state.CustomerContacts = flattenCloneCustomerContacts(*cloneProps.CustomerContacts)
	}

	// Bad - another example with different flatten method
	if props.Items != nil { // want "AZNR006:"
		result.Items = flattenItems(*props.Items)
	}

	// Bad - flatten method with multiple parameters but dereferencing in call
	if data.Config != nil { // want "AZNR006:"
		state.Configuration = flattenConfiguration(*data.Config, otherParam)
	}

	// Bad - nil check before flatten call (no dereferencing but still should be inside flatten)
	if cloneProps.CustomerContacts != nil { // want "AZNR006:"
		state.CustomerContacts = flattenPointerCustomerContacts(cloneProps.CustomerContacts)
	}
}

type SomeState struct {
	CustomerContacts interface{}
	OtherField       interface{}
	Items            interface{}
	Configuration    interface{}
}

type SomeProps struct {
	CustomerContacts *interface{}
	OtherField       *string
	Items            *interface{}
}

type Props struct {
	Items *interface{}
}

type Data struct {
	Config *interface{}
}

var props Props
var data Data
var result SomeState
var otherParam string
var someData *interface{}

func flattenCloneCustomerContacts(contacts interface{}) interface{} {
	return contacts
}

func flattenItems(items interface{}) interface{} {
	return items
}

func flattenConfiguration(config interface{}, other string) interface{} {
	return config
}

func flattenPointerCustomerContacts(contacts *interface{}) interface{} {
	if contacts == nil {
		return nil
	}
	return *contacts
}

func processData(data interface{}) {
	// some processing
}
