package AZNR001

import (
	"go/ast"
	"sort"
	"strings"

	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemafields"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemainfo"
	"github.com/qixialu/azurerm-linter/passes/util"
	"golang.org/x/tools/go/analysis"
)

const analyzerName = "AZNR001"

const Doc = `check for Schema field ordering

The AZNR001 analyzer reports cases of schemas where fields are not ordered correctly.

Schema fields should be ordered as follows:
1. Any fields that make up the resource's ID, with the last user specified segment 
   (usually the resource's name) first. (e.g. 'name' then 'resource_group_name', 
   or 'name' then 'parent_resource_id')
2. The 'location' field.
3. Required fields, sorted alphabetically.
4. Optional fields, sorted alphabetically.
5. Computed fields, sorted alphabetically.`

var skipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var skipFileSuffix = []string{"_test.go", "registration.go"}

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  Doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip specified packages
	pkgPath := pass.Pkg.Path()
	for _, skip := range skipPackages {
		if strings.Contains(pkgPath, skip) {
			return nil, nil
		}
	}

	schemaInfo := schemainfo.GetSchemaInfo()

	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename

		if !changedlines.IsFileChanged(filename) {
			continue
		}

		// Skip non-newly added files
		if !changedlines.IsNewFile(filename) {
			continue
		}

		skipFile := false
		for _, skip := range skipFileSuffix {
			if strings.HasSuffix(filename, skip) {
				skipFile = true
				break
			}
		}
		if skipFile {
			continue
		}

		nestedSchemas := schemafields.FindNestedSchemas(f)

		ast.Inspect(f, func(n ast.Node) bool {
			// Look for composite literals that might be schema maps
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if it's a schema map
			if !schemafields.IsSchemaMap(comp) {
				return true
			}

			// Extract schema fields
			fields := schemafields.ExtractFromCompositeLit(pass, comp, schemaInfo)
			if len(fields) == 0 {
				return true
			}

			// Check if this is a nested schema (inside Elem field)
			// Nested schemas should not be checked for ID fields and location ordering
			isNested := nestedSchemas[comp]

			// Check for ordering issues
			expectedOrder, issue := checkOrderingIssues(fields, isNested)
			if issue != "" {
				actualOrder := make([]string, len(fields))
				for i, f := range fields {
					actualOrder[i] = f.Name
				}
				pass.Reportf(comp.Pos(), "%s: %s\nExpected order:\n  %s\nActual order:\n  %s\n",
					analyzerName, issue,
					util.FixedCode(strings.Join(expectedOrder, ", ")),
					util.IssueLine(strings.Join(actualOrder, ", ")))
			}

			return true
		})
	}

	return nil, nil
}

// checkOrderingIssues validates field ordering according to the rules:
// For top-level schemas:
//  1. name must come before resource_group_name, resource_group_name must come before location
//     (required fields can be in between)
//  2. optional+computed fields must be continuous (no required fields in between)
//
// For nested schemas:
//   - required fields must come before optional fields
//   - optional fields must come before computed fields
func checkOrderingIssues(fields []schemafields.SchemaField, isNested bool) ([]string, string) {
	if len(fields) == 0 {
		return nil, ""
	}

	expectedOrder := getExpectedOrder(fields, isNested)
	return expectedOrder, validateOrder(fields, expectedOrder, isNested)
}

func getExpectedOrder(fields []schemafields.SchemaField, isNested bool) []string {
	fieldMap := make(map[string]schemafields.SchemaField)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	var result []string

	if !isNested {
		// Track which special fields exist and are required
		specialRequiredFields := make(map[string]bool)
		for _, field := range fields {
			if field.Name == "name" || field.Name == "resource_group_name" || field.Name == "location" {
				if field.Required {
					specialRequiredFields[field.Name] = true
				}
			}
		}

		// First, add required special fields in the correct order
		for _, fieldName := range []string{"name", "resource_group_name", "location"} {
			if specialRequiredFields[fieldName] {
				result = append(result, fieldName)
			}
		}

		// Then categorize and add other fields
		var requiredFields []string
		var optionalFields []string
		var computedFields []string

		for _, field := range fields {
			// Skip special required fields as they're already added
			if (field.Name == "name" || field.Name == "resource_group_name" || field.Name == "location") && field.Required {
				continue
			}

			switch {
			case field.Required:
				requiredFields = append(requiredFields, field.Name)
			case field.Optional:
				optionalFields = append(optionalFields, field.Name)
			case field.Computed:
				computedFields = append(computedFields, field.Name)
			}
		}

		// Required fields maintain their original order
		result = append(result, requiredFields...)

		// Optional and computed fields are sorted alphabetically
		sort.Strings(optionalFields)
		sort.Strings(computedFields)

		result = append(result, optionalFields...)
		result = append(result, computedFields...)
	} else {
		// Nested schema
		var requiredFields []string
		var optionalFields []string
		var computedFields []string

		for _, field := range fields {
			switch {
			case field.Required:
				requiredFields = append(requiredFields, field.Name)
			case field.Optional:
				optionalFields = append(optionalFields, field.Name)
			case field.Computed:
				computedFields = append(computedFields, field.Name)
			}
		}

		sort.Strings(requiredFields)
		sort.Strings(optionalFields)
		sort.Strings(computedFields)

		result = append(result, requiredFields...)
		result = append(result, optionalFields...)
		result = append(result, computedFields...)
	}

	return result
}

func validateOrder(fields []schemafields.SchemaField, expectedOrder []string, isNested bool) string {
	if len(fields) != len(expectedOrder) {
		// Skip if len is not equal, it might because the schema is defined in another package, except commonschema
		return ""
	}

	if !isNested {
		// For top-level schemas, check relative positions of name, resource_group_name, location
		fieldMap := make(map[string]int)
		for i, field := range fields {
			fieldMap[field.Name] = i
		}

		nameIdx, hasName := fieldMap["name"]
		rgIdx, hasRG := fieldMap["resource_group_name"]
		locIdx, hasLoc := fieldMap["location"]

		if hasName && hasRG && nameIdx > rgIdx {
			return "'resource_group_name' field must come after 'name' field"
		}
		if hasRG && hasLoc && rgIdx > locIdx {
			return "'location' field must come after 'resource_group_name' field"
		}
		if hasName && hasLoc && nameIdx > locIdx {
			return "'location' field must come after 'name' field"
		}

		// Check optional and computed fields are in correct alphabetical order
		// Build a list of optional and computed fields in their actual order
		var optionalActual []string
		var computedActual []string

		for _, field := range fields {
			if field.Name == "name" || field.Name == "resource_group_name" || field.Name == "location" {
				continue
			}

			isOptional := field.Optional
			isComputed := field.Computed && !field.Optional && !field.Required

			if isOptional {
				optionalActual = append(optionalActual, field.Name)
			} else if isComputed {
				computedActual = append(computedActual, field.Name)
			}
		}

		optionalSorted := true
		for i := 0; i < len(optionalActual)-1; i++ {
			if optionalActual[i] > optionalActual[i+1] {
				optionalSorted = false
				break
			}
		}

		computedSorted := true
		for i := 0; i < len(computedActual)-1; i++ {
			if computedActual[i] > computedActual[i+1] {
				computedSorted = false
				break
			}
		}

		if !optionalSorted || !computedSorted {
			return "schema fields are not in the correct order"
		}

		return ""
	}

	// For nested schemas, check exact order
	actualOrder := make([]string, len(fields))
	for i, f := range fields {
		actualOrder[i] = f.Name
	}

	for i := range actualOrder {
		if actualOrder[i] != expectedOrder[i] {
			return "schema fields are not in the correct order"
		}
	}

	return ""
}
