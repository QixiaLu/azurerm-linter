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
var skipFileSuffix = []string{"_test.go", "registration.go"}

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		schemainfo.Analyzer,
	},
	Run: run,
}

type schemaField struct {
	name     string
	required bool
	optional bool
	computed bool
	position int // original position in schema
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

	// Cache modelFieldMapping and ID fields per file for performance
	modelMappingCache := make(map[*ast.File]map[string]string)
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

		// Build modelFieldMapping once per file
		modelFieldMapping, ok := modelMappingCache[f]
		if !ok {
			modelFieldMapping = buildModelFieldMapping(f)
			modelMappingCache[f] = modelFieldMapping
		}

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
			fields := extractSchemaFields(pass, comp, schemaInfo)
			if len(fields) == 0 {
				return true
			}

			// Check if this is a nested schema (inside Elem field)
			// Nested schemas should not be checked for ID fields and location ordering
			isNested := isNestedSchema(comp, f)

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
				actualOrder[i] = f.name
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

func extractSchemaFields(pass *analysis.Pass, smap *ast.CompositeLit, schemaInfo *schemainfo.SchemaInfo) []schemaField {
	var fields []schemaField

	for i, elt := range smap.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		// Get field name
		fieldName := astutils.ExprStringValue(kv.Key)
		if fieldName == nil {
			continue
		}

		field := schemaField{
			name:     *fieldName,
			position: i,
		}

		// Try to parse the value - it could be either:
		// 1. A composite literal: &schema.Schema{...}
		// 2. A function call: commonschema.ResourceGroupName()

		switch v := kv.Value.(type) {
		case *ast.CompositeLit:
			// Direct schema definition
			schema := schema.NewSchemaInfo(v, pass.TypesInfo)
			field.required = schema.Schema.Required
			field.optional = schema.Schema.Optional
			field.computed = schema.Schema.Computed

		case *ast.CallExpr:
			// Function call - try multiple resolution strategies
			resolved := false

			// Strategy 1: Try to get from schemainfo cache (for cross-package functions)
			if selExpr, ok := v.Fun.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
					if obj := pass.TypesInfo.Uses[pkgIdent]; obj != nil {
						if pkgName, ok := obj.(*types.PkgName); ok {
							funcKey := pkgName.Imported().Path() + "." + selExpr.Sel.Name
							if props, ok := schemaInfo.Functions[funcKey]; ok {
								// Use schema info from the loaded package
								field.required = props.Required
								field.optional = props.Optional
								field.computed = props.Computed
								resolved = true
							}
						}
					}
				}
			}

			// Strategy 2: Try to resolve from same-package function definition
			if !resolved {
				if resolvedSchema := resolveSchemaFromFuncCall(pass, v); resolvedSchema != nil {
					field.required = resolvedSchema.Schema.Required
					field.optional = resolvedSchema.Schema.Optional
					field.computed = resolvedSchema.Schema.Computed
					resolved = true
				}
			}
		default:
			// Unknown type, skip
			continue
		}

		fields = append(fields, field)
	}

	return fields
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

		// Look for Create methods (resources) or Read methods (data sources)
		if funcDecl.Name == nil {
			return true
		}
		methodName := funcDecl.Name.Name

		// For data sources, look in Read; for resources, look in Create
		if isDataSource {
			if !strings.Contains(methodName, "Read") {
				return true
			}
		} else {
			if !strings.Contains(methodName, "Create") {
				return true
			}
		}

		// Build variable resolver (scans function body only once)
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
			// This handles cases like: apiId := fmt.Sprintf(...), id := NewApiID(..., apiId)
			for _, field := range fields {
				if field == "" {
					return true // Skip this SetID call
				}
			}

			for _, field := range fields {
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
	parentIDFields    map[string]string // parent ID variable -> schema field name (e.g., routeTableId -> route_table_id)
}

func newVariableResolver(funcDecl *ast.FuncDecl, modelFieldMapping map[string]string) *variableResolver {
	resolver := &variableResolver{
		varToField:        make(map[string]string),
		idVarAssignments:  make(map[string][]string),
		modelFieldMapping: modelFieldMapping,
		parentIDFields:    make(map[string]string),
	}

	// Single pass: build all mappings
	ast.Inspect(funcDecl, func(n ast.Node) bool {
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

			// 2. Check if RHS is a Parse*ID call (parent ID pattern)
			// e.g., routeTableId, err := virtualwans.ParseHubRouteTableID(d.Get("route_table_id").(string))
			if call, ok := rhs.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if strings.HasPrefix(sel.Sel.Name, "Parse") && strings.HasSuffix(sel.Sel.Name, "ID") {
						// Extract the field name from the Parse call argument
						if len(call.Args) > 0 {
							if fieldName := extractFieldNameFromArg(call.Args[0], modelFieldMapping); fieldName != "" {
								resolver.parentIDFields[lhsIdent.Name] = fieldName
							}
						}
						continue
					}
				}
			}

			// 3. Check if RHS is a New*ID call
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}

			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if strings.HasPrefix(sel.Sel.Name, "New") && strings.HasSuffix(sel.Sel.Name, "ID") {
					// Check if any arguments are selector expressions (parent ID fields)
					hasParentIDFields := false
					var parentIDVar string

					for _, arg := range call.Args {
						if selExpr, ok := arg.(*ast.SelectorExpr); ok {
							if ident, ok := selExpr.X.(*ast.Ident); ok {
								// This is a parent ID pattern (e.g., routeTableId.SubscriptionId)
								hasParentIDFields = true
								parentIDVar = ident.Name
								break
							}
						}
					}

					if hasParentIDFields {
						// Parent resource ID pattern: only extract direct fields, ignore parent ID fields
						fields := resolver.resolveFieldsFromArgsIgnoringParent(call.Args, parentIDVar)

						// Add parent ID field to the list
						if parentField, ok := resolver.parentIDFields[parentIDVar]; ok {
							fields = append(fields, parentField)
						}

						resolver.idVarAssignments[lhsIdent.Name] = fields
					} else {
						// Regular pattern: resolve all fields
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

	for _, arg := range args {
		fieldName := r.resolveFieldName(arg)

		// Skip duplicates
		if seen[fieldName] {
			continue
		}
		seen[fieldName] = true

		// Add field name (including empty string for unresolvable fields)
		// Empty strings will be detected later to skip the entire ID extraction
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

// resolveFieldsFromArgsIgnoringParent resolves arguments but ignores fields from parent ID object
// This is used for parent resource ID patterns where ID construction uses fields from another parsed ID
// e.g. azurerm_virtual_hub_route_table_route
func (r *variableResolver) resolveFieldsFromArgsIgnoringParent(args []ast.Expr, parentIDVar string) []string {
	var fields []string
	seen := make(map[string]bool)

	for _, arg := range args {
		// Skip selector expressions on parent ID (e.g., routeTableId.SubscriptionId)
		if selExpr, ok := arg.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok && ident.Name == parentIDVar {
				continue // Ignore parent ID fields
			}
		}

		if fieldName := r.resolveFieldName(arg); fieldName != "" {
			if !seen[fieldName] {
				seen[fieldName] = true
				fields = append(fields, fieldName)
			}
		}
	}

	return fields
}

// extractFieldsFromIDArg extracts field names from the SetID argument
// This is a method of variableResolver for cleaner access to mappings
func (r *variableResolver) extractFieldsFromIDArg(expr ast.Expr) []string {
	// Case 1: Direct identifier (variable holding the ID)
	if ident, ok := expr.(*ast.Ident); ok {
		if assignedFields, ok := r.idVarAssignments[ident.Name]; ok {
			return assignedFields
		}
	}

	// Case 2: Direct New*ID call - use resolver to handle all arguments uniformly
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if strings.HasPrefix(sel.Sel.Name, "New") && strings.HasSuffix(sel.Sel.Name, "ID") {
				return r.resolveFieldsFromArgs(call.Args)
			}
		}
	}

	// Case 3: Method call on ID (e.g., id.ID())
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "ID" {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Look up the ID variable's field assignments
					if assignedFields, ok := r.idVarAssignments[ident.Name]; ok {
						return assignedFields
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

// buildModelFieldMapping builds a map from Go field names to tfschema tag values
// by parsing struct type definitions in the file
func buildModelFieldMapping(node *ast.File) map[string]string {
	mapping := make(map[string]string)

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type declarations
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		// Process each spec in the declaration
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check if it's a struct type
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Only process structs with "Model" in their name
			if !strings.Contains(typeSpec.Name.Name, "Model") {
				continue
			}

			// Extract field mappings from struct fields
			for _, field := range structType.Fields.List {
				if field.Tag == nil {
					continue
				}

				// Parse the struct tag
				tagValue := field.Tag.Value
				// Remove backticks
				tagValue = strings.Trim(tagValue, "`")

				// Look for tfschema tag
				if strings.Contains(tagValue, "tfschema:") {
					// Extract the tfschema value
					if schemaName := extractTFSchemaTag(tagValue); schemaName != "" {
						// Map each field name to its tfschema value
						for _, name := range field.Names {
							mapping[name.Name] = schemaName
						}
					}
				}
			}
		}

		return true
	})

	return mapping
}

// extractTFSchemaTag extracts the value from a tfschema struct tag
// Input: `tfschema:"resource_group_name"`
// Output: resource_group_name
func extractTFSchemaTag(tag string) string {
	// Look for tfschema:"value"
	start := strings.Index(tag, "tfschema:\"")
	if start == -1 {
		return ""
	}
	start += len("tfschema:\"")

	end := strings.Index(tag[start:], "\"")
	if end == -1 {
		return ""
	}

	return tag[start : start+end]
}

func getExpectedOrder(fields []schemaField, idFields []string, isNested bool) []string {
	// Create a map for quick lookup of field properties
	fieldMap := make(map[string]schemaField)
	for _, field := range fields {
		fieldMap[field.name] = field
	}

	// Categorize fields
	var idFieldsList []string
	var locationField []string
	var requiredFields []string
	var optionalFields []string
	var computedFields []string

	for _, field := range fields {
		// Check if it's an ID field (only for top-level schemas)
		if !isNested {
			isIDField := false
			for _, idField := range idFields {
				if field.name == idField {
					// If ID field is computed-only, treat it as computed field instead
					if field.computed && !field.optional && !field.required {
						computedFields = append(computedFields, field.name)
					} else {
						idFieldsList = append(idFieldsList, field.name)
					}
					isIDField = true
					break
				}
			}
			if isIDField {
				continue
			}

			// Check if it's location (only for top-level schemas)
			if field.name == "location" {
				// If location is computed-only, treat it as computed field instead
				if field.computed && !field.optional && !field.required {
					computedFields = append(computedFields, field.name)
				} else {
					locationField = append(locationField, field.name)
				}
				continue
			}
		}

		// Categorize by type (for all schemas including nested)
		if field.computed && !field.optional && !field.required {
			computedFields = append(computedFields, field.name)
		} else if field.required {
			requiredFields = append(requiredFields, field.name)
		} else if field.optional {
			optionalFields = append(optionalFields, field.name)
		} else {
			// Default to optional if no flags set
			optionalFields = append(optionalFields, field.name)
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

// isNestedSchema checks if a schema map is nested within an Elem field definition.
// Nested schemas (e.g., those inside Elem of a List/Set) shouldn't be checked for
// resource ID field ordering requirements, as they are not top-level resource schemas.
func isNestedSchema(comp *ast.CompositeLit, file *ast.File) bool {
	// Track current node in AST traversal
	var isNested bool

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for key-value expressions (schema field definitions)
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if the key is "Elem"
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Elem" {
			return true
		}

		// Check if our schema map is within this Elem value
		ast.Inspect(kv.Value, func(n2 ast.Node) bool {
			if n2 == comp {
				isNested = true
				return false
			}
			return true
		})

		return !isNested // stop searching if we found it
	})

	return isNested
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
