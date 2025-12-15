package schemafields

import (
	"go/ast"
	"go/types"

	"github.com/bflad/tfproviderlint/helper/astutils"
	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemainfo"
	"golang.org/x/tools/go/analysis"
)

// SchemaField represents a field in a Terraform schema with its properties
type SchemaField struct {
	Name     string
	Required bool
	Optional bool
	Computed bool
	position int
}

// ExtractFromCompositeLit extracts schema fields from a map[string]*schema.Schema composite literal
// parentField is the name of the parent field (e.g., "Elem") if this schema is nested, empty string otherwise
func ExtractFromCompositeLit(pass *analysis.Pass, smap *ast.CompositeLit, schemaInfo *schemainfo.SchemaInfo) []SchemaField {
	var fields []SchemaField

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

		field := SchemaField{
			Name:     *fieldName,
			position: i,
		}

		// Try to parse the value - it could be either:
		// 1. A composite literal: &schema.Schema{...}
		// 2. A function call: commonschema.ResourceGroupName()

		switch v := kv.Value.(type) {
		case *ast.CompositeLit:
			// Direct schema definition
			schema := schema.NewSchemaInfo(v, pass.TypesInfo)
			field.Required = schema.Schema.Required
			field.Optional = schema.Schema.Optional
			field.Computed = schema.Schema.Computed

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
								field.Required = props.Required
								field.Optional = props.Optional
								field.Computed = props.Computed
								resolved = true
							}
						}
					}
				}
			}

			// Strategy 2: Try to resolve from same-package function definition
			if !resolved {
				if resolvedSchema := resolveSchemaFromFuncCall(pass, v); resolvedSchema != nil {
					field.Required = resolvedSchema.Schema.Required
					field.Optional = resolvedSchema.Schema.Optional
					field.Computed = resolvedSchema.Schema.Computed
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

// findNestedSchemas finds all schema maps that are nested within Elem fields
func FindNestedSchemas(file *ast.File) map[*ast.CompositeLit]bool {
	nestedSchemas := make(map[*ast.CompositeLit]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for Elem key-value expressions
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Elem" {
			return true
		}

		// Mark all schema maps within this Elem value as nested
		ast.Inspect(kv.Value, func(n2 ast.Node) bool {
			comp, ok := n2.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if it's a schema map
			mapType, ok := comp.Type.(*ast.MapType)
			if !ok {
				return true
			}

			if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
				return true
			}

			starExpr, ok := mapType.Value.(*ast.StarExpr)
			if !ok {
				return true
			}

			selExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || selExpr.Sel.Name != "Schema" {
				return true
			}

			// This is a schema map inside Elem
			nestedSchemas[comp] = true
			return true
		})

		return true
	})

	return nestedSchemas
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
