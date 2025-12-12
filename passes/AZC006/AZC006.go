package AZC006

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"github.com/bflad/tfproviderlint/helper/astutils"
	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/helpers/modelmapping"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemafields"
	"github.com/qixialu/azurerm-linter/passes/schemainfo"
	"golang.org/x/tools/go/analysis"
)

const analyzerName = "AZC006"

const Doc = `check for Schema field ordering

The AZC006 analyzer reports cases of schemas where fields are not ordered correctly.

Schema fields should be ordered as follows:
1. Any fields that make up the resource's ID, with the last user specified segment 
   (usually the resource's name) first. (e.g. 'name' then 'resource_group_name', 
   or 'name' then 'parent_resource_id')
2. The 'location' field.
3. Required fields, sorted alphabetically.
4. Optional fields, sorted alphabetically.
5. Computed fields, sorted alphabetically.`

// skipPackages lists package path patterns to skip during analysis
var skipPackages = []string{"/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var skipFileSuffix = []string{
	"_test.go",
	"registration.go",
	"ip_groups_data_source.go", // Resource Group Name is used to get id, but name is also used later on. Unable to use following pattern to order idFields
}

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		schemainfo.Analyzer,
	},
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip specified packages
	pkgPath := pass.Pkg.Path()
	for _, skip := range skipPackages {
		if strings.Contains(pkgPath, skip) {
			return nil, nil
		}
	}

	schemaInfo := pass.ResultOf[schemainfo.Analyzer].(*schemainfo.SchemaInfo)

	idFieldsCache := make(map[*ast.File][]string)

	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename

		if !changedlines.IsFileChanged(filename) {
			continue
		}

		for _, skip := range skipFileSuffix {
			if strings.HasSuffix(filename, skip) {
				continue
			}
		}

		// Check if it's a resource, data source, or helper file
		isResourceFile := strings.HasSuffix(filename, "_resource.go")
		isDataSourceFile := strings.HasSuffix(filename, "_data_source.go")

		modelFieldMapping := modelmapping.BuildForFile(pass, f)
		nestedSchemas := schemafields.FindNestedSchemas(f)

		// Extract ID fields once per file (only for resource and data source files)
		var idFields []string
		if isResourceFile || isDataSourceFile {
			idFieldsCached, ok := idFieldsCache[f]
			if !ok {
				isDataSource := isDataSourceFile
				idFieldsCached = extractIDFieldsFromFile(f, modelFieldMapping, isDataSource)
				idFieldsCache[f] = idFieldsCached
			}
			idFields = idFieldsCached
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Look for composite literals that might be schema maps
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if it's a map literal (map[string]*schema.Schema or map[string]*pluginsdk.Schema)
			mapType, ok := comp.Type.(*ast.MapType)
			if !ok {
				return true
			}

			// Check if key is string
			if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
				return true
			}

			// Check if value is *schema.Schema or *pluginsdk.Schema
			starExpr, ok := mapType.Value.(*ast.StarExpr)
			if !ok {
				return true
			}

			selExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || selExpr.Sel.Name != "Schema" {
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

			// Determine if we should check ID fields and location
			// Skip for: helper files, nested schemas
			shouldCheckIDAndLocation := (isResourceFile || isDataSourceFile) && !isNested

			// For schemas that don't check ID/location, use empty ID fields list
			var effectiveIDFields []string
			if shouldCheckIDAndLocation {
				if len(idFields) == 0 {
					// Skip - unable to extract ID fields from SetID call
					//
					// This can happen in cases where the ID is constructed from API responses
					// rather than directly from schema fields. For example:
					//
					//   name := d.Get("name").(string)
					//   storageAccount, err := parse.ParseOptionallyVersionedNestedItemID(d.Get("managed_storage_account_id").(string))
					//   ...
					//   keyVaultIdRaw, err := keyVaultsClient.KeyVaultIDFromBaseUrl(ctx, subscriptionResourceId, storageAccount.KeyVaultBaseUrl)
					//   ...
					//   keyVaultBaseUri, err := keyVaultsClient.BaseUriForKeyVault(ctx, *keyVaultId)
					//   ...
					//   read, err := client.GetSasDefinition(ctx, baseUri, storageAccount.Name, name)
					//   sasId, err := parse.SasDefinitionID(*read.ID)
					//   d.SetId(sasId.ID())
					//
					// In this pattern, the ID comes from the API response (*read.ID), not from
					// a New*ID() constructor. To support this, we would need to:
					//   1. Trace sasId back to the Parse*ID call
					//   2. Trace *read.ID back to the API call (client.GetSasDefinition)
					//   3. Analyze each API call parameter to find schema fields
					//   4. Handle multiple levels of variable references and transformations
					//
					// This adds significant complexity for a relatively rare pattern. Most resources
					// use the standard New*ID() pattern which is already supported.
					//
					// For these cases, manual review is recommended to ensure proper field ordering.
					fmt.Printf("[%s] Skipping %s: unable to extract ID fields from SetID call\n", analyzerName, filename)
					return true
				}
				effectiveIDFields = idFields
			} else {
				// Helper files or nested schemas: don't check ID fields or location
				effectiveIDFields = []string{}
			}

			// Get expected order (isNested flag is now replaced by checking effectiveIDFields)
			expectedOrder := getExpectedOrder(fields, effectiveIDFields, !shouldCheckIDAndLocation)
			actualOrder := make([]string, len(fields))
			for i, f := range fields {
				actualOrder[i] = f.Name
			}

			// Check if order is correct
			if !areOrdersEqual(actualOrder, expectedOrder) {
				pos := pass.Fset.Position(comp.Pos())
				if changedlines.ShouldReport(pos.Filename, pos.Line) {
					pass.Reportf(comp.Pos(), "%s: schema fields are not in the correct order\nExpected order:\n  %s\nActual order:\n  %s",
						analyzerName,
						strings.Join(expectedOrder, ", "),
						strings.Join(actualOrder, ", "))
				}
			}

			return true
		})
	}

	return nil, nil
}

// resolveSchemaFromFuncCall attempts to resolve schema info from a function call
func resolveSchemaFromFuncCall(pass *analysis.Pass, call *ast.CallExpr) *schema.SchemaInfo {
	// Get the function selector
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// Get the function object from TypesInfo
	funcObj := pass.TypesInfo.Uses[sel.Sel]
	if funcObj == nil {
		return nil
	}

	// Get the function declaration
	funcDecl := findFuncDecl(pass, funcObj)
	if funcDecl == nil {
		return nil
	}

	// Look for return statement that returns a schema
	var returnedSchema *ast.CompositeLit
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if ret, ok := n.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
			// Check if the return value is a composite literal or unary expression (&schema.Schema{...})
			var expr ast.Expr = ret.Results[0]

			// Handle &schema.Schema{...}
			if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op.String() == "&" {
				expr = unary.X
			}

			// Check if it's a composite literal
			if comp, ok := expr.(*ast.CompositeLit); ok {
				returnedSchema = comp
				return false // Stop inspection
			}
		}
		return true
	})

	if returnedSchema == nil {
		return nil
	}

	// Parse the returned schema
	return schema.NewSchemaInfo(returnedSchema, pass.TypesInfo)
}

// findFuncDecl finds the function declaration for a given function object
func findFuncDecl(pass *analysis.Pass, funcObj interface{}) *ast.FuncDecl {
	// Convert to types.Object to get position
	obj, ok := funcObj.(types.Object)
	if !ok {
		return nil
	}

	objPos := obj.Pos()

	// Search in all files in the pass (same package)
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				if funcDecl.Name != nil {
					// Get the position of this function declaration
					declPos := pass.Fset.Position(funcDecl.Pos())
					targetPos := pass.Fset.Position(objPos)

					// Compare file and position
					if declPos.Filename == targetPos.Filename &&
						declPos.Line == targetPos.Line {
						return funcDecl
					}
				}
			}
		}
	}

	return nil
}

// extractIDFieldsFromFile dynamically extracts ID field names from Create or Read functions
// by finding metadata.SetID() or d.SetId() calls and tracing back the ID construction
// For resources, it looks in Create methods; for data sources, it looks in Read methods
func extractIDFieldsFromFile(node *ast.File, modelFieldMapping map[string]string, isDataSource bool) []string {
	var idFields []string
	seen := make(map[string]bool)

	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if funcDecl.Name == nil {
			return true
		}
		methodName := funcDecl.Name.Name

		if isDataSource {
			if !strings.Contains(methodName, "Read") {
				return true
			}
		} else {
			if !strings.Contains(methodName, "Create") {
				return true
			}
		}

		// Build variable resolver
		resolver := newVariableResolver(funcDecl, modelFieldMapping)

		// Walk the function body looking for SetID/SetId calls
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isSetIDCall(call) {
				return true
			}

			if len(call.Args) == 0 {
				return true
			}

			// Use resolver to extract field names from the ID argument
			fields := resolver.extractFieldsFromIDArg(call.Args[0])

			// Skip if any field is empty (unresolvable)
			for _, field := range fields {
				if field == "" {
					return true // Skip this SetID call
				}
				if !seen[field] {
					seen[field] = true
					idFields = append(idFields, field)
				}
			}

			return true
		})

		return true
	})

	return idFields

}

// variableResolver resolves variables to field names using AST analysis
// It scans the function body only once to build all necessary mappings
type variableResolver struct {
	varToField        map[string]string   // variable name -> field name
	idVarAssignments  map[string][]string // ID variable -> list of field names
	modelFieldMapping map[string]string
}

func newVariableResolver(funcDecl *ast.FuncDecl, modelFieldMapping map[string]string) *variableResolver {
	resolver := &variableResolver{
		varToField:        make(map[string]string),
		idVarAssignments:  make(map[string][]string),
		modelFieldMapping: modelFieldMapping,
	}

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) {
				break
			}

			lhsIdent, ok := assign.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}

			// 1. Check if RHS is d.Get("field_name") or model.FieldName
			if fieldName := extractFieldNameFromArg(rhs, modelFieldMapping); fieldName != "" {
				resolver.varToField[lhsIdent.Name] = fieldName
				continue
			}

			if call, ok := rhs.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					// Only extract fields from New*ID or Parse*ID calls
					funcName := sel.Sel.Name
					if strings.HasPrefix(funcName, "New") && strings.HasSuffix(funcName, "ID") ||
						strings.HasPrefix(funcName, "Parse") && strings.HasSuffix(funcName, "ID") {
						fields := resolver.resolveFieldsFromArgs(call.Args)
						resolver.idVarAssignments[lhsIdent.Name] = fields
					}
				}
			}
		}

		return true
	})

	return resolver
}

// resolveFieldsFromArgs resolves a list of arguments to field names
func (r *variableResolver) resolveFieldsFromArgs(args []ast.Expr) []string {
	var fields []string
	seen := make(map[string]bool)

	for i, arg := range args {
		fieldName := r.resolveFieldName(arg)

		// Special case: skip subscription ID (first argument only)
		if i == 0 && fieldName == "" {
			// Check if it's a selector expression (e.g., meta.SubscriptionId)
			if selExpr, ok := arg.(*ast.SelectorExpr); ok {
				selectorName := selExpr.Sel.Name
				if strings.Contains(strings.ToLower(selectorName), "subscription") {
					continue
				}
			}
			// Check if it's an identifier variable (e.g., subscriptionId)
			if ident, ok := arg.(*ast.Ident); ok {
				if strings.Contains(strings.ToLower(ident.Name), "subscription") {
					continue
				}
			}
		}

		if seen[fieldName] {
			continue
		}
		seen[fieldName] = true

		fields = append(fields, fieldName)
	}
	return fields
}

// resolveFieldName recursively resolves an expression to a field name
func (r *variableResolver) resolveFieldName(expr ast.Expr) string {
	// Try direct extraction (d.Get("field") or model.Field)
	if fieldName := extractFieldNameFromArg(expr, r.modelFieldMapping); fieldName != "" {
		return fieldName
	}

	// If it's a variable reference, look it up
	if ident, ok := expr.(*ast.Ident); ok {
		if fieldName, ok := r.varToField[ident.Name]; ok {
			return fieldName
		}
	}

	return ""
}

// extractFieldsFromIDArg extracts field names from the SetID argument
func (r *variableResolver) extractFieldsFromIDArg(expr ast.Expr) []string {
	// Case 1: Direct identifier (variable holding the ID)
	if ident, ok := expr.(*ast.Ident); ok {
		if assignedFields, ok := r.idVarAssignments[ident.Name]; ok {
			return assignedFields
		}
		if fieldName, ok := r.varToField[ident.Name]; ok {
			return []string{fieldName}
		}
	}

	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			// Case 2: Method call on ID (e.g., id.ID())
			if sel.Sel.Name == "ID" {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Look up the ID variable's field assignments
					if assignedFields, ok := r.idVarAssignments[ident.Name]; ok {
						return assignedFields
					}
					if assignedFields, ok := r.varToField[ident.Name]; ok {
						return []string{assignedFields}
					}
				}
			}
		}
	}

	return nil
}

// isSetIDCall checks if a call expression is a SetID or SetId call
func isSetIDCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if sel.Sel.Name != "SetID" && sel.Sel.Name != "SetId" {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "metadata" || ident.Name == "meta" || ident.Name == "d"
}

// extractFieldNameFromArg extracts field names from both:
// 1. d.Get("field_name").(string) - traditional SDK
// 2. model.FieldName - typed SDK
func extractFieldNameFromArg(expr ast.Expr, modelFieldMapping map[string]string) string {
	// Try to extract from d.Get("field_name")
	if fieldName := extractFieldNameFromDGet(expr); fieldName != "" {
		return fieldName
	}

	// Try to extract from model.FieldName
	return extractFieldNameFromModelAccess(expr, modelFieldMapping)
}

// extractFieldNameFromDGet extracts the field name from d.Get("field_name").(string) expressions
func extractFieldNameFromDGet(expr ast.Expr) string {
	// Handle type assertions: d.Get("field_name").(string)
	if typeAssert, ok := expr.(*ast.TypeAssertExpr); ok {
		expr = typeAssert.X
	}

	// Check if it's a call expression
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	// Check if it's d.Get
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok || selExpr.Sel.Name != "Get" {
		return ""
	}

	// Check if the receiver is 'd'
	if ident, ok := selExpr.X.(*ast.Ident); !ok || ident.Name != "d" {
		return ""
	}

	// Extract the string literal argument
	if len(callExpr.Args) > 0 {
		if fieldName := astutils.ExprStringValue(callExpr.Args[0]); fieldName != nil {
			return *fieldName
		}
	}

	return ""
}

// extractFieldNameFromModelAccess extracts field name from model.FieldName
// using the modelFieldMapping to get the correct tfschema tag value
func extractFieldNameFromModelAccess(expr ast.Expr, modelFieldMapping map[string]string) string {
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	// Look up the field name in the mapping
	goFieldName := selExpr.Sel.Name
	if schemaFieldName, ok := modelFieldMapping[goFieldName]; ok {
		return schemaFieldName
	}

	// Fallback: convert field name from PascalCase to snake_case
	// e.g., ResourceGroupName -> resource_group_name
	return toSnakeCase(goFieldName)
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func getExpectedOrder(fields []schemafields.SchemaField, idFields []string, isNested bool) []string {
	// Create a map for quick lookup of field properties
	fieldMap := make(map[string]schemafields.SchemaField)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	// Categorize fields
	var idFieldsList []string
	var locationField []string
	var requiredFields []string
	var optionalFields []string
	var computedFields []string

	idFieldsSet := make(map[string]bool)
	if !isNested {
		for i := len(idFields) - 1; i >= 0; i-- {
			idField := idFields[i]
			if f, ok := fieldMap[idField]; ok {
				if !(f.Computed && !f.Optional && !f.Required) {
					idFieldsList = append(idFieldsList, idField)
					idFieldsSet[idField] = true
				}
			}
		}
	}

	for _, field := range fields {
		// Check if it's an ID field (already processed above)
		if idFieldsSet[field.Name] {
			continue
		}

		// Check if it's location (only for top-level schemas)
		if !isNested && field.Name == "location" {
			locationField = append(locationField, field.Name)
			continue
		}

		// Categorize by type (for all schemas including nested)
		if field.Computed && !field.Optional && !field.Required {
			computedFields = append(computedFields, field.Name)
		} else if field.Required {
			requiredFields = append(requiredFields, field.Name)
		} else if field.Optional {
			optionalFields = append(optionalFields, field.Name)
		} else {
			// Default to optional if no flags set
			optionalFields = append(optionalFields, field.Name)
		}
	}

	// Sort other categories alphabetically
	sort.Strings(requiredFields)
	sort.Strings(optionalFields)
	sort.Strings(computedFields)

	// Combine in the expected order
	var result []string
	result = append(result, idFieldsList...)
	result = append(result, locationField...)
	result = append(result, requiredFields...)
	result = append(result, optionalFields...)
	result = append(result, computedFields...)

	return result
}

func areOrdersEqual(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}
