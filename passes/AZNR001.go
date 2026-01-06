package passes

import (
	"go/ast"
	"sort"
	"strings"

	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR001Doc = `check for Schema field ordering

The AZNR001 analyzer reports cases of schemas where fields are not ordered correctly.

When git filter is applied, it only works on newly created files.

Schema fields should be ordered as follows:
1. Any fields that make up the resource's ID, with the last user specified segment 
   (usually the resource's name) first. (e.g. 'name' then 'resource_group_name', 
   or 'name' then 'parent_resource_id')
2. The 'location' field.
3. Required fields, sorted alphabetically.
   (Since it might contain fields made up the resource's ID, these require special ordering. 
   And the linter currently cannot get those fields. 
   The sorted rule for required properties at top level is skipped)
4. Optional fields, sorted alphabetically.
5. Computed fields, sorted alphabetically.
6. Tags field`

const aznr001Name = "AZNR001"

var aznr001SkipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var aznr001SkipFileSuffix = []string{"_test.go", "registration.go"}

var AZNR001Analyzer = &analysis.Analyzer{
	Name:     aznr001Name,
	Doc:      AZNR001Doc,
	Run:      runAZNR001,
	Requires: []*analysis.Analyzer{inspect.Analyzer, schema.CommonAnalyzer},
}

func runAZNR001(pass *analysis.Pass) (interface{}, error) {
	// Skip specified packages
	pkgPath := pass.Pkg.Path()
	for _, skip := range aznr001SkipPackages {
		if strings.Contains(pkgPath, skip) {
			return nil, nil
		}
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}
	commonSchemaInfo, ok := pass.ResultOf[schema.CommonAnalyzer].(*schema.CommonSchemaInfo)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.CompositeLit)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		// Apply filename filtering
		filename := pass.Fset.Position(comp.Pos()).Filename
		if !loader.IsNewFile(filename) {
			return
		}

		skipFile := false
		for _, skip := range aznr001SkipFileSuffix {
			if strings.HasSuffix(filename, skip) {
				skipFile = true
				break
			}
		}
		if skipFile {
			return
		}

		// Check if it's a schema map
		if !helper.IsSchemaMap(comp) {
			return
		}

		// Extract schema fields
		fields := schema.ExtractFromCompositeLit(pass, comp, commonSchemaInfo)
		if len(fields) == 0 {
			return
		}

		// Check if this schema is nested within an Elem field
		isNested := false
		for _, f := range pass.Files {
			fPos := pass.Fset.Position(f.Pos())
			if fPos.Filename == filename {
				isNested = helper.IsNestedSchemaMap(f, comp)
				break
			}
		}

		// Check for ordering issues
		expectedOrder, issue := checkAZNR001OrderingIssues(fields, isNested)
		if issue != "" {
			actualOrder := make([]string, len(fields))
			for i, f := range fields {
				actualOrder[i] = f.Name
			}
			pass.Reportf(comp.Pos(), "%s: %s\nExpected order:\n  %s\nActual order:\n  %s\n",
				aznr001Name, issue,
				helper.FixedCode(strings.Join(expectedOrder, ", ")),
				helper.IssueLine(strings.Join(actualOrder, ", ")))
		}
	})

	return nil, nil
}

func checkAZNR001OrderingIssues(fields []helper.SchemaFieldInfo, isNested bool) ([]string, string) {
	if len(fields) == 0 {
		return nil, ""
	}

	expectedOrder := getAZNR001ExpectedOrder(fields, isNested)
	return expectedOrder, validateAZNR001Order(fields, expectedOrder, isNested)
}

func getAZNR001ExpectedOrder(fields []helper.SchemaFieldInfo, isNested bool) []string {
	fieldMap := make(map[string]helper.SchemaFieldInfo)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	var result []string

	if !isNested {
		// Track which special fields exist and are required
		specialRequiredFields := make(map[string]bool)
		var locationIsComputed bool
		for _, field := range fields {
			if field.Name == "name" || field.Name == "resource_group_name" || field.Name == "location" {
				if field.SchemaInfo != nil && field.SchemaInfo.Schema.Required {
					specialRequiredFields[field.Name] = true
				}
				if field.Name == "location" && field.SchemaInfo != nil && field.SchemaInfo.Schema.Computed {
					locationIsComputed = true
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
		var tagsField string

		for _, field := range fields {
			// Skip special required fields as they're already added
			if (field.Name == "name" || field.Name == "resource_group_name" || field.Name == "location") && field.SchemaInfo != nil && field.SchemaInfo.Schema.Required {
				continue
			}

			// Handle tags field separately
			if field.Name == "tags" {
				tagsField = field.Name
				continue
			}

			// Skip location if it's computed (will be added at the beginning of computed fields)
			if field.Name == "location" && locationIsComputed {
				continue
			}

			if field.SchemaInfo != nil {
				switch {
				case field.SchemaInfo.Schema.Required:
					requiredFields = append(requiredFields, field.Name)
				case field.SchemaInfo.Schema.Optional:
					optionalFields = append(optionalFields, field.Name)
				case field.SchemaInfo.Schema.Computed:
					computedFields = append(computedFields, field.Name)
				}
			}
		}

		// Required fields maintain their original order
		result = append(result, requiredFields...)

		// Optional and computed fields are sorted alphabetically
		sort.Strings(optionalFields)
		sort.Strings(computedFields)

		result = append(result, optionalFields...)

		// Add location at the beginning of computed fields if it's computed
		if locationIsComputed {
			result = append(result, "location")
		}
		result = append(result, computedFields...)

		// Add tags field at the end if it exists
		if tagsField != "" {
			result = append(result, tagsField)
		}
	} else {
		// Nested schema
		var requiredFields []string
		var optionalFields []string
		var computedFields []string

		for _, field := range fields {
			if field.SchemaInfo != nil {
				switch {
				case field.SchemaInfo.Schema.Required:
					requiredFields = append(requiredFields, field.Name)
				case field.SchemaInfo.Schema.Optional:
					optionalFields = append(optionalFields, field.Name)
				case field.SchemaInfo.Schema.Computed:
					computedFields = append(computedFields, field.Name)
				}
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

func validateAZNR001Order(fields []helper.SchemaFieldInfo, expectedOrder []string, isNested bool) string {
	if len(fields) != len(expectedOrder) {
		// Skip if len is not equal, it happens when it's failed to extract field's properties;
		// it might because the schema is defined in another package, except commonschema
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

		// Check if location is computed
		locationIsComputed := false
		if hasLoc {
			locField := fields[locIdx]
			if locField.SchemaInfo != nil && locField.SchemaInfo.Schema.Computed && !locField.SchemaInfo.Schema.Required && !locField.SchemaInfo.Schema.Optional {
				locationIsComputed = true
			}
		}

		// Only check location ordering if it's not computed
		if !locationIsComputed {
			if hasName && hasRG && nameIdx > rgIdx {
				return "'resource_group_name' field must come after 'name' field"
			}
			if hasRG && hasLoc && rgIdx > locIdx {
				return "'location' field must come after 'resource_group_name' field"
			}
			if hasName && hasLoc && nameIdx > locIdx {
				return "'location' field must come after 'name' field"
			}
		} else {
			// If location is computed, only check name and resource_group_name ordering
			if hasName && hasRG && nameIdx > rgIdx {
				return "'resource_group_name' field must come after 'name' field"
			}
		}

		// Check optional and computed fields are in correct alphabetical order
		// Build a list of optional and computed fields in their actual order
		var optionalActual []string
		var computedActual []string
		var tagsIdx = -1
		var locationComputedIdx = -1

		for i, field := range fields {
			if field.Name == "tags" {
				tagsIdx = i
				continue
			}

			// Check if location is computed
			if field.Name == "location" && field.SchemaInfo != nil && field.SchemaInfo.Schema.Computed {
				locationComputedIdx = i
				continue
			}

			if field.Name == "name" || field.Name == "resource_group_name" || (field.Name == "location" && field.SchemaInfo != nil && field.SchemaInfo.Schema.Required) {
				continue
			}

			if field.SchemaInfo != nil {
				isOptional := field.SchemaInfo.Schema.Optional
				isComputed := field.SchemaInfo.Schema.Computed && !field.SchemaInfo.Schema.Optional && !field.SchemaInfo.Schema.Required

				if isOptional {
					optionalActual = append(optionalActual, field.Name)
				} else if isComputed {
					computedActual = append(computedActual, field.Name)
				}
			}
		}

		// Check if tags field is at the end (if it exists)
		if tagsIdx != -1 && tagsIdx != len(fields)-1 {
			return "'tags' field must be at the end of the schema"
		}

		// Check if computed location is at the beginning of computed fields
		if locationComputedIdx != -1 && len(computedActual) > 0 {
			// Find the index of the first computed field (excluding location)
			firstComputedIdx := -1
			for i, field := range fields {
				if field.Name == "location" {
					continue
				}
				if field.SchemaInfo != nil && field.SchemaInfo.Schema.Computed && !field.SchemaInfo.Schema.Optional && !field.SchemaInfo.Schema.Required {
					firstComputedIdx = i
					break
				}
			}
			if firstComputedIdx != -1 && locationComputedIdx > firstComputedIdx {
				return "'location' field must be at the beginning of computed fields"
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
