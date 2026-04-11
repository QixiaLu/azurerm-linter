package schema

const (
	TypeInvalid = iota
	TypeBool
	TypeInt
	TypeString
	TypeList
	TypeMap
	TypeSet
)

type SchemaValidateFunc func(interface{}, string) ([]string, []error)

type Resource struct {
	Schema map[string]*Schema
}

type ResourceData struct{}

type Schema struct {
	Type          int
	Required      bool
	Optional      bool
	Computed      bool
	ForceNew      bool
	Sensitive     bool
	Elem          interface{}
	ValidateFunc  SchemaValidateFunc
	AtLeastOneOf  []string
	ExactlyOneOf  []string
	ConflictsWith []string
}
