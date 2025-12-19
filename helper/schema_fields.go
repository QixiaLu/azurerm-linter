package helpers

import (
	"go/ast"
	"go/types"

	"github.com/bflad/tfproviderlint/helper/astutils"
	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/internal/commonschemainfo"
	"golang.org/x/tools/go/analysis"
)

// SchemaFieldInfo represents a field in a Terraform schema with its schema information
type SchemaFieldInfo struct {
	Name       string
	SchemaInfo *schema.SchemaInfo
	Position   int
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
func ExtractFromCompositeLit(pass *analysis.Pass, smap *ast.CompositeLit, commonSchemaInfo *commonschemainfo.SchemaInfo) []SchemaFieldInfo {
	fields := make([]SchemaFieldInfo, 0, len(smap.Elts))

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

		// Resolve schema info from the value
		var resolvedSchema *schema.SchemaInfo
		switch v := kv.Value.(type) {
		case *ast.CompositeLit:
			// Direct schema definition: &schema.Schema{...}
			resolvedSchema = schema.NewSchemaInfo(v, pass.TypesInfo)
		case *ast.CallExpr:
			// Function call: try to resolve
			resolvedSchema = resolveSchemaInfoFromCall(pass, v, commonSchemaInfo)
		default:
			// Unknown type, skip
			continue
		}

		fields = append(fields, SchemaFieldInfo{
			Name:       *fieldName,
			SchemaInfo: resolvedSchema,
			Position:   i,
		})
	}

	return fields
}

// IsNestedSchemaMap checks if a schema map CompositeLit is nested within an Elem field
// It uses position-based checking which is fast with early termination
func IsNestedSchemaMap(file *ast.File, schemaLit *ast.CompositeLit) bool {
	var isNested bool

	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if this is an Elem key
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Elem" {
			return true
		}

		// Check if our schemaLit is within this Elem value's range
		if schemaLit.Pos() >= kv.Value.Pos() && schemaLit.End() <= kv.Value.End() {
			isNested = true
			return false // Found it, stop searching immediately
		}

		return true
	})

	return isNested
}

// resolveSchemaInfoFromCall resolves schema info from a function call
// It tries cross-package cache first, then same-package resolution
func resolveSchemaInfoFromCall(pass *analysis.Pass, call *ast.CallExpr, commonSchemaInfo *commonschemainfo.SchemaInfo) *schema.SchemaInfo {
	// Strategy 1: Try to get from commonSchemaInfo cache (for cross-package functions)
	if selExpr, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
			if obj := pass.TypesInfo.Uses[pkgIdent]; obj != nil {
				if pkgName, ok := obj.(*types.PkgName); ok {
					funcKey := pkgName.Imported().Path() + "." + selExpr.Sel.Name
					if cachedSchemaInfo, ok := commonSchemaInfo.Functions[funcKey]; ok {
						return cachedSchemaInfo
					}
				}
			}
		}
	}

	// Strategy 2: Try to resolve from same-package function definition
	return resolveSchemaFromFuncCall(pass, call)
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
