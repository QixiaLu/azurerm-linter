package modelmapping

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// BuildForFile builds model field mapping for a specific file
// {modelFieldName: schemaName}
func BuildForFile(pass *analysis.Pass, file *ast.File) map[string]string {
	mapping := make(map[string]string)

	if modelType := findModelTypeFromModelObject(pass, file); modelType != nil {
		extractFieldsFromStructType(modelType, mapping)
	}

	return mapping
}

// findModelTypeFromModelObject finds the Model type from ModelObject{} method
func findModelTypeFromModelObject(pass *analysis.Pass, file *ast.File) *types.Struct {
	var modelType *types.Struct

	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			return true
		}

		if funcDecl.Name.Name != "ModelObject" {
			return true
		}

		ast.Inspect(funcDecl.Body, func(n2 ast.Node) bool {
			ret, ok := n2.(*ast.ReturnStmt)
			if !ok || len(ret.Results) == 0 {
				return true
			}

			var expr ast.Expr = ret.Results[0]

			if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op.String() == "&" {
				expr = unary.X
			}

			comp, ok := expr.(*ast.CompositeLit)
			if !ok {
				return true
			}

			typeInfo := pass.TypesInfo.TypeOf(comp)
			if typeInfo != nil {
				if ptr, ok := typeInfo.(*types.Pointer); ok {
					typeInfo = ptr.Elem()
				}

				if named, ok := typeInfo.(*types.Named); ok {
					if structType, ok := named.Underlying().(*types.Struct); ok {
						modelType = structType
						return false
					}
				}
			}

			if selExpr, ok := comp.Type.(*ast.SelectorExpr); ok {
				if resolvedType := resolveImportedModelType(pass, selExpr); resolvedType != nil {
					modelType = resolvedType
					return false
				}
			}

			return true
		})

		if modelType != nil {
			return false
		}

		return true
	})

	return modelType
}

func resolveImportedModelType(pass *analysis.Pass, selExpr *ast.SelectorExpr) *types.Struct {
	pkgIdent, ok := selExpr.X.(*ast.Ident)
	if !ok {
		return nil
	}

	typeName := selExpr.Sel.Name

	pkgObj := pass.TypesInfo.Uses[pkgIdent]
	if pkgObj == nil {
		return nil
	}

	pkgName, ok := pkgObj.(*types.PkgName)
	if !ok {
		return nil
	}

	importedPkg := pkgName.Imported()
	if importedPkg == nil {
		return nil
	}

	typeObj := importedPkg.Scope().Lookup(typeName)
	if typeObj == nil {
		return nil
	}

	typeNameObj, ok := typeObj.(*types.TypeName)
	if !ok {
		return nil
	}

	namedType := typeNameObj.Type()
	if named, ok := namedType.(*types.Named); ok {
		if structType, ok := named.Underlying().(*types.Struct); ok {
			return structType
		}
	}

	return nil
}

func extractFieldsFromStructType(structType *types.Struct, mapping map[string]string) {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		if schemaName := parseTFSchemaTag(tag); schemaName != "" {
			mapping[field.Name()] = schemaName
		}
	}
}

func parseTFSchemaTag(tag string) string {
	const prefix = `tfschema:"`
	start := strings.Index(tag, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)

	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return ""
	}

	return tag[start : start+end]
}
