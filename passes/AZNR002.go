package passes

import (
	"go/ast"
	"go/types"
	"log"
	"strings"

	"github.com/bflad/tfproviderlint/helper/astutils"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/loader"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR002Doc = `check that top-level updatable properties are handled in Update function

The AZNR002 analyzer checks that all updatable properties (not marked as ForceNew)
are properly handled in the Update function for typed resources.

If git filter enabled, this rule only applies on newly created file.

This analyzer will be skipped if a helper function is utilized to handle the update.

For typed resources, this means checking for metadata.ResourceData.HasChange("property_name").

Example violation:
  // In Arguments()
  "display_name": {
      Type:     pluginsdk.TypeString,
      Required: true,
      // No ForceNew - this is updatable
  }

  // In Update() - missing HasChange check
  func (r Resource) Update() sdk.ResourceFunc {
      return sdk.ResourceFunc{
          Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
              // Missing: if metadata.ResourceData.HasChange("display_name") { ... }
              return nil
          },
      }
  }

Valid usage:
  func (r Resource) Update() sdk.ResourceFunc {
      return sdk.ResourceFunc{
          Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
              if metadata.ResourceData.HasChange("display_name") {
                  props.DisplayName = pointer.To(config.DisplayName)
              }
              return nil
          },
      }
  }`

const aznr002Name = "AZNR002"

var aznr002SkipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}

var AZNR002Analyzer = &analysis.Analyzer{
	Name:     aznr002Name,
	Doc:      AZNR002Doc,
	Run:      runAZNR002,
	Requires: []*analysis.Analyzer{inspect.Analyzer, schema.CommonAnalyzer, buildssa.Analyzer},
}

func runAZNR002(pass *analysis.Pass) (interface{}, error) {
	// Skip specified packages
	pkgPath := pass.Pkg.Path()
	for _, skip := range aznr002SkipPackages {
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

	// Find all typed resources in this package
	typedResources := findTypedResourcesWithUpdate(pass, inspector)

	// Analyze each typed resource
	for _, resource := range typedResources {
		// Step 1: Extract updatable properties from schema
		// TODO: Could get from internal provider instead of AST Parsing if this rule is included under internal/tools in AzureRM
		updatableProps := extractUpdatableProperties(pass, resource, commonSchemaInfo)

		// Step 2: Find handled properties in Update()
		handledProps := findHandledPropertiesInUpdate(pass, resource)

		// Step 3: Report missing properties
		reportMissingProperties(pass, resource, updatableProps, handledProps)
	}

	return nil, nil
}

// findTypedResourcesWithUpdate identifies all typed resources in the package
func findTypedResourcesWithUpdate(pass *analysis.Pass, inspector *inspector.Inspector) []*helper.TypedResourceInfo {
	var resources []*helper.TypedResourceInfo

	// First pass: find type declarations that implement sdk.ResourceWithUpdate
	nodeFilter := []ast.Node{(*ast.GenDecl)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return
		}

		fileName := pass.Fset.Position(genDecl.Pos()).Filename
		if !loader.IsFileChanged(fileName) || !loader.IsNewFile(fileName) {
			return
		}

		if !strings.HasSuffix(fileName, "_resource.go") {
			return
		}

		// Check for interface implementation: var _ sdk.ResourceWithUpdate = TypeName{}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Check if this is implementing sdk.ResourceWithUpdate
			if !helper.IsResourceWithUpdateInterface(valueSpec.Type) {
				continue
			}

			// Get the resource type name
			if len(valueSpec.Values) == 0 {
				continue
			}

			var resourceTypeName string
			switch v := valueSpec.Values[0].(type) {
			case *ast.CompositeLit:
				if ident, ok := v.Type.(*ast.Ident); ok {
					resourceTypeName = ident.Name
				}
			}

			if resourceTypeName == "" {
				continue
			}

			// Find the file containing this resource
			for _, file := range pass.Files {
				filePos := pass.Fset.Position(file.Pos()).Filename
				if filePos != fileName {
					continue
				}

				// Create fully populated TypedResourceInfo - all parsing done inside constructor
				resource := helper.NewTypedResourceInfo(resourceTypeName, file, pass.TypesInfo)

				// Only add if it's complete (has all required components)
				if resource.ModelStruct != nil && resource.ArgumentsFunc != nil && resource.UpdateFunc != nil {
					resources = append(resources, resource)
				}
			}
		}
	})

	return resources
}

// extractUpdatableProperties extracts all updatable properties from the schema
func extractUpdatableProperties(pass *analysis.Pass, resource *helper.TypedResourceInfo, commonSchemaInfo *schema.CommonSchemaInfo) map[string]string {
	updatableProps := make(map[string]string)

	funcDecl := resource.ArgumentsFunc
	if funcDecl == nil || funcDecl.Body == nil {
		return updatableProps
	}

	// Look for return statement with map[string]*pluginsdk.Schema
	var schemaMap *ast.CompositeLit
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		if compLit, ok := ret.Results[0].(*ast.CompositeLit); ok {
			if helper.IsSchemaMap(compLit) {
				schemaMap = compLit
				return false
			}
		}

		return true
	})
	if schemaMap == nil || astutils.ExprValue(schemaMap) == nil {
		return updatableProps
	}

	fields := schema.ExtractFromCompositeLit(pass, schemaMap, commonSchemaInfo)
	// Filter updatable properties (not Computed, not ForceNew)
	for _, field := range fields {
		if field.SchemaInfo != nil && !field.SchemaInfo.Schema.Computed && !field.SchemaInfo.Schema.ForceNew {
			var modelFieldName string
			for fieldName, tfschema := range resource.ModelFieldToTFSchema {
				if tfschema == field.Name {
					modelFieldName = fieldName
					break
				}
			}
			updatableProps[field.Name] = modelFieldName
		}
	}

	return updatableProps
}

// findHandledPropertiesInUpdate finds all properties handled in Update function
func findHandledPropertiesInUpdate(pass *analysis.Pass, resource *helper.TypedResourceInfo) map[string]bool {
	handledProps := make(map[string]bool)

	if resource.UpdateFunc == nil || resource.UpdateFunc.Body == nil {
		return handledProps
	}

	// Find the returned sdk.ResourceFunc
	var updateFuncBody *ast.BlockStmt
	ast.Inspect(resource.UpdateFunc.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		// Look for sdk.ResourceFunc{ Func: func(...) { ... } }
		compLit, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			return true
		}

		for _, elt := range compLit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Func" {
				if funcLit, ok := kv.Value.(*ast.FuncLit); ok {
					updateFuncBody = funcLit.Body
					return false
				}
			}
		}

		return true
	})

	if updateFuncBody == nil {
		return handledProps
	}

	// Get the model struct type name
	modelTypeName := resource.ModelName

	// Pattern 1: Check if model/config is passed to helper functions
	// If detected, skip this resource as properties are likely handled in helper
	if detectModelPassedToHelper(updateFuncBody, modelTypeName, resource.TypesInfo) {
		return handledProps
	}

	// Single pass: inspect all nodes and check both patterns
	ast.Inspect(updateFuncBody, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				methodName := sel.Sel.Name

				// Pattern 2 & 3: Check ResourceData method calls (HasChange/HasChanges/Get)
				if methodName == "HasChange" || methodName == "HasChanges" || methodName == "Get" {
					if isResourceDataMethod(sel, resource.TypesInfo) {
						if methodName == "Get" && len(node.Args) > 0 {
							// Pattern 3: Get("property_name")
							if propName := astutils.ExprStringValue(node.Args[0]); propName != nil {
								handledProps[*propName] = true
							}
						} else if methodName == "HasChange" || methodName == "HasChanges" {
							// Pattern 2: HasChange("prop") or HasChanges("prop1", "prop2")
							for _, arg := range node.Args {
								if propName := astutils.ExprStringValue(arg); propName != nil {
									handledProps[*propName] = true
								}
							}
						}
					}
				}
			}

		case *ast.SelectorExpr:
			// Pattern 4: state.FieldName or config.FieldName
			// Check if the field name matches any of our model fields
			fieldName := node.Sel.Name
			if tfschemaName, ok := resource.ModelFieldToTFSchema[fieldName]; ok {
				// This is a field from our model struct being accessed
				// Now verify the base is likely a model variable by checking with TypesInfo
				if resource.TypesInfo != nil {
					if typ := resource.TypesInfo.TypeOf(node.X); typ != nil {
						// Remove pointer if present
						if ptr, ok := typ.(*types.Pointer); ok {
							typ = ptr.Elem()
						}
						// Check if it's a named type matching our model
						if named, ok := typ.(*types.Named); ok {
							if obj := named.Obj(); obj != nil && obj.Name() == modelTypeName {
								handledProps[tfschemaName] = true
							}
						}
					}
				}
			}
		}

		return true
	})

	return handledProps
}

// isResourceDataMethod checks if a selector expression is a method call on ResourceData type
func isResourceDataMethod(sel *ast.SelectorExpr, typesInfo *types.Info) bool {
	if typesInfo == nil {
		return false
	}

	// Get the type of the selector's base expression
	typ := typesInfo.TypeOf(sel.X)
	if typ == nil {
		return false
	}

	// Check if it's a pointer to ResourceData
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// Check the type name contains "ResourceData"
	return strings.Contains(typ.String(), "ResourceData")
}

// detectModelPassedToHelper checks if model/config variable is passed to helper functions
// Returns true if expand/map/flatten functions are called with model variables at TOP LEVEL (not inside if/for blocks)
// e.g. "automanage_configuration_resource.go": expandConfigurationProfile(model) - should skip
// counter-example "spring_cloud_gateway_resource.go": if HasChange { expandGatewayResponseCacheProperties(model) } - should NOT skip
func detectModelPassedToHelper(body *ast.BlockStmt, modelTypeName string, typesInfo *types.Info) bool {
	// Only check top-level statements in the function body
	for _, stmt := range body.List {
		// Skip if statement, for statement, switch statement - these are conditional
		switch stmt.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.RangeStmt:
			continue
		}

		found := false
		// Check assignments and expression statements at top level
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if any argument is the model variable (by type)
			for _, arg := range call.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					// Use TypesInfo to check if this variable is of model type
					if typ := typesInfo.TypeOf(ident); typ != nil {
						if ptr, ok := typ.(*types.Pointer); ok {
							typ = ptr.Elem()
						}
						// Check if it's a named type matching our model
						if named, ok := typ.(*types.Named); ok {
							if obj := named.Obj(); obj != nil && obj.Name() == modelTypeName {
								found = true
								return false
							}
						}
					}
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// reportMissingProperties reports properties that are updatable but not handled
func reportMissingProperties(pass *analysis.Pass, resource *helper.TypedResourceInfo, updatableProps map[string]string, handledProps map[string]bool) {
	var missingProps []string

	for propName := range updatableProps {
		if !handledProps[propName] {
			missingProps = append(missingProps, propName)
		}
	}

	// Skip if handledProps len is 0, it's most likely delegated to a helper func
	if len(missingProps) == 0 || len(handledProps) == 0 {
		if len(handledProps) == 0 {
			pos := pass.Fset.Position(resource.UpdateFunc.Pos())
			log.Printf("%s:%d: %s: Skipping resource %q - the update implementation is delegated to a helper function",
				pos.Filename, pos.Line, aznr002Name, resource.ResourceTypeName)
		}
		return
	}

	// Sort for consistent output
	for i := 0; i < len(missingProps); i++ {
		for j := i + 1; j < len(missingProps); j++ {
			if missingProps[i] > missingProps[j] {
				missingProps[i], missingProps[j] = missingProps[j], missingProps[i]
			}
		}
	}

	// Report at the Update function
	if resource.UpdateFunc != nil {
		pass.Reportf(resource.UpdateFunc.Pos(),
			"%s: resource has updatable properties not handled in Update function: `%s`. If they are non-updatable, mark them as %s in Arguments() schema\n",
			aznr002Name,
			helper.IssueLine(strings.Join(missingProps, ", ")),
			helper.FixedCode("ForceNew: true"))
	}
}
