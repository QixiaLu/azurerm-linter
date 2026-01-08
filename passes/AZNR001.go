package passes

import (
	"go/ast"
	"sort"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	passesschema "github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR001Doc = `check for top-level Schema field ordering

The AZNR001 analyzer reports cases of schemas where fields are not ordered correctly.

When git filter is applied, it only works on newly created files.

Schema fields should be ordered as follows:

1. Required fields in their original order
2. 'resource_group_name' must come before 'location' if both are required
3. Optional fields, sorted alphabetically (unless they appear before required fields)
4. Computed fields, sorted alphabetically (with 'location' first if computed)
5. 'tags' field must be at the end

Special cases:
- Schemas with 'name' field which is optional are skipped
- Schemas with optional+computed+ForceNew '*_id' or '*_name' fields are skipped
- If optional fields appear before required fields, their original order is preserved
  (This happens when some resources have optional fields as part of the resource ID components)
- Fields that are part of the resource ID (including 'name') require manual verification of their order
- Nested schemas are not validated by this rule`

const aznr001Name = "AZNR001"

const (
	fieldName              = "name"
	fieldResourceGroupName = "resource_group_name"
	fieldLocation          = "location"
	fieldTags              = "tags"
)

var aznr001SkipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var aznr001FileSuffix = []string{"_resource.go", "_data_source.go"}

func isComputedOnly(info *schema.SchemaInfo) bool {
	if info == nil {
		return false
	}
	return info.Schema.Computed && !info.Schema.Optional && !info.Schema.Required
}

func isSorted(fields []string) bool {
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] > fields[i+1] {
			return false
		}
	}
	return true
}

func hasOptionalNameWithExactlyOneOf(fields []helper.SchemaFieldInfo) bool {
	for _, field := range fields {
		if field.Name == fieldName && field.SchemaInfo != nil {
			// Check if it's optional
			if field.SchemaInfo.Schema.Optional {
				return true
			}
		}
	}
	return false
}

var AZNR001Analyzer = &analysis.Analyzer{
	Name:     aznr001Name,
	Doc:      AZNR001Doc,
	Run:      runAZNR001,
	Requires: []*analysis.Analyzer{inspect.Analyzer, passesschema.CommonAnalyzer},
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
	commonSchemaInfo, ok := pass.ResultOf[passesschema.CommonAnalyzer].(*passesschema.CommonSchemaInfo)
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

		// Only process files with specific suffixes
		validFile := false
		for _, suffix := range aznr001FileSuffix {
			if strings.HasSuffix(filename, suffix) {
				validFile = true
				break
			}
		}
		if !validFile {
			return
		}

		// Check if it's a schema map
		if !helper.IsSchemaMap(comp) {
			return
		}

		// Extract schema fields
		fields := passesschema.ExtractFromCompositeLit(pass, comp, commonSchemaInfo)
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
		// Skip nested schema for now, as this rule seems to apply for the top-level only
		// The nested schema order is not consistent
		if isNested {
			return
		}

		// Check for ordering issues
		expectedOrder, issue := checkAZNR001OrderingIssues(fields)
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

func checkAZNR001OrderingIssues(fields []helper.SchemaFieldInfo) ([]string, string) {
	if len(fields) == 0 {
		return nil, ""
	}

	// Skip if name field is optional + ExactlyOneOf (e.g., name/name_regex pattern (image_data_source.go))
	// Skip if there's alternative id field (e.g., _id with optional + forceNew (cosmosdb_sql_role_definition_resource.go))
	if hasOptionalNameWithExactlyOneOf(fields) {
		return nil, ""
	}

	expectedOrder := getAZNR001ExpectedOrder(fields)
	return expectedOrder, validateAZNR001Order(fields, expectedOrder)
}

func getAZNR001ExpectedOrder(fields []helper.SchemaFieldInfo) []string {
	// Track if location is computed
	var locationIsComputed bool
	for _, field := range fields {
		if field.Name == fieldLocation && field.SchemaInfo != nil && field.SchemaInfo.Schema.Computed {
			locationIsComputed = true
			break
		}
	}

	// Check if there are optional fields before required fields
	// If so, preserve the original order (don't sort optional fields)
	// This is because some resources have optional ID fields at the beginning (before required fields)
	// and we want to preserve their original order for those special cases
	hasOptionalBeforeRequired := false
	lastOptionalIdx := -1
	firstRequiredIdx := -1
	for i, field := range fields {
		if field.SchemaInfo != nil {
			if field.SchemaInfo.Schema.Optional && lastOptionalIdx == -1 {
				lastOptionalIdx = i
			}
			if field.SchemaInfo.Schema.Required && firstRequiredIdx == -1 {
				firstRequiredIdx = i
			}
		}
	}
	if lastOptionalIdx != -1 && firstRequiredIdx != -1 && lastOptionalIdx < firstRequiredIdx {
		hasOptionalBeforeRequired = true
	}

	// Collect fields by type, preserving original order for required fields
	var requiredFields []string
	var optionalFields []string
	var computedFields []string
	var tagsField string

	for _, field := range fields {
		// Handle tags field separately
		if field.Name == fieldTags {
			tagsField = field.Name
			continue
		}

		// Skip location if it's computed (will be added at the beginning of computed fields)
		if field.Name == fieldLocation && locationIsComputed {
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

	// Only fix resource_group_name < location order if both are required
	rgPos := -1
	locPos := -1
	for i, name := range requiredFields {
		if name == fieldResourceGroupName {
			rgPos = i
		}
		if name == fieldLocation {
			locPos = i
		}
	}
	// Swap if location comes before resource_group_name
	if rgPos != -1 && locPos != -1 && locPos < rgPos {
		requiredFields[rgPos], requiredFields[locPos] = requiredFields[locPos], requiredFields[rgPos]
	}

	// Build result: required (keep original order with rg<loc fix) → optional → computed → tags
	// Note: If optional fields appear before required fields, keep original order (don't sort)
	var result []string
	result = append(result, requiredFields...)

	// Only sort optional fields if they don't appear before required fields
	if !hasOptionalBeforeRequired {
		sort.Strings(optionalFields)
	}
	result = append(result, optionalFields...)

	// Add location at the beginning of computed fields if it's computed
	if locationIsComputed {
		result = append(result, fieldLocation)
	}
	sort.Strings(computedFields)
	result = append(result, computedFields...)

	// Add tags field at the end if it exists
	if tagsField != "" {
		result = append(result, tagsField)
	}

	return result
}

func validateAZNR001Order(fields []helper.SchemaFieldInfo, expectedOrder []string) string {
	if len(fields) != len(expectedOrder) {
		// Skip if len is not equal, it happens when it's failed to extract field's properties;
		// it might because the schema is defined in another package, except commonschema
		return ""
	}

	// Compare actual order with expected order
	actualOrder := make([]string, len(fields))
	for i, field := range fields {
		actualOrder[i] = field.Name
	}

	// Check if actual order matches expected order
	for i := 0; i < len(actualOrder); i++ {
		if actualOrder[i] != expectedOrder[i] {
			return "schema fields are not in the expected order, please double check the order as mentioned in /guide-new-resource.md or /guide-new-data-source.md"
		}
	}

	return ""
}
