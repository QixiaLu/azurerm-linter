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

// IsSchemaMap checks if a composite literal is a map[string]*schema.Schema or map[string]*pluginsdk.Schema
func IsSchemaMap(comp *ast.CompositeLit) bool {
	// Check if it's a map literal
	mapType, ok := comp.Type.(*ast.MapType)
	if !ok {
		return false
	}

	// Check if key is string
	if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
		return false
	}

	// Check if value is *schema.Schema or *pluginsdk.Schema
	starExpr, ok := mapType.Value.(*ast.StarExpr)
	if !ok {
		return false
	}

	selExpr, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok || selExpr.Sel.Name != "Schema" {
		return false
	}

	return true
}

// ExtractFromCompositeLit extracts schema fields from a map[string]*schema.Schema composite literal
// parentField is the name of the parent field (e.g., "Elem") if this schema is nested, empty string otherwise
func ExtractFromCompositeLit(pass *analysis.Pass, smap *ast.CompositeLit, schemaInfo *schemainfo.SchemaInfo) []SchemaField {
	fields := make([]SchemaField, 0, len(smap.Elts))

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
			found := false

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
								found = true
							}
						}
					}
				}
			}

			// Strategy 2: Try to resolve from same-package function definition (if not found in cache)
			if !found {
				if resolvedSchema := resolveSchemaFromFuncCall(pass, v); resolvedSchema != nil {
					field.Required = resolvedSchema.Schema.Required
					field.Optional = resolvedSchema.Schema.Optional
					field.Computed = resolvedSchema.Schema.Computed
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
	var funcObj types.Object

	// Handle both selector expressions (pkg.Function) and identifiers (Function)
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		// Cross-package function call like commonschema.ResourceGroupName()
		funcObj = pass.TypesInfo.Uses[fun.Sel]
	case *ast.Ident:
		// Same-package function call like metadataSchema()
		funcObj = pass.TypesInfo.Uses[fun]
	default:
		return nil
	}

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
func findFuncDecl(pass *analysis.Pass, funcObj types.Object) *ast.FuncDecl {
	obj, ok := funcObj.(*types.Func)
	if !ok {
		return nil
	}

	pos := obj.Pos()

	for _, file := range pass.Files {
		// Check if the position is within this file's range
		if pos < file.Pos() || pos > file.End() {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Match by function name position
			if funcDecl.Name.Pos() == pos {
				return funcDecl
			}
		}
	}

	return nil
}
