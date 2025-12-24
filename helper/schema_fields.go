package helper

import (
	"go/ast"
	"go/types"
	"go/token"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
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

// FindFuncDecl finds the function declaration for a given function object
func FindFuncDecl(pass *analysis.Pass, funcObj types.Object) *ast.FuncDecl {
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

// GetResourceSchemaFromElem extracts the &schema.Resource{...} composite literal from an Elem field
// Returns nil if Elem is not a pointer to a Resource composite literal
func GetResourceSchemaFromElem(elemKV *ast.KeyValueExpr) *ast.CompositeLit {
	if unary, ok := elemKV.Value.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		if compLit, ok := unary.X.(*ast.CompositeLit); ok {
			return compLit
		}
	}

	return nil
}

// GetNestedSchemaMap extracts the Schema field value from a Resource composite literal
// Returns nil if the Schema field is not found or is not a composite literal
func GetNestedSchemaMap(resourceSchema *ast.CompositeLit) *ast.CompositeLit {
	var nestedSchemaMap *ast.CompositeLit
	for _, fld := range resourceSchema.Elts {
		fieldKV, ok := fld.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := fieldKV.Key.(*ast.Ident); ok && ident.Name == "Schema" {
			if compLit, ok := fieldKV.Value.(*ast.CompositeLit); ok {
				nestedSchemaMap = compLit
			}
			break
		}
	}
	return nestedSchemaMap
}
