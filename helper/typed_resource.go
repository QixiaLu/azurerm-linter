package helper

import (
	"go/ast"
	"go/types"
	"strings"
)

// TypedResourceInfo represents gathered information about a typed Terraform resource
type TypedResourceInfo struct {
	ResourceTypeName     string
	ModelName            string
	ModelStruct          *ast.StructType
	ArgumentsFunc        *ast.FuncDecl
	AttributesFunc       *ast.FuncDecl
	CreateFunc           *ast.FuncDecl
	ReadFunc             *ast.FuncDecl
	UpdateFunc           *ast.FuncDecl
	UpdateFuncBody       *ast.BlockStmt
	DeleteFunc           *ast.FuncDecl
	TypesInfo            *types.Info
	ModelFieldToTFSchema map[string]string // model struct field name -> tfschema tag name
}

// NewTypedResourceInfo creates a TypedResourceInfo by parsing a typed resource from file
func NewTypedResourceInfo(resourceTypeName string, file *ast.File, info *types.Info) *TypedResourceInfo {
	result := &TypedResourceInfo{
		ResourceTypeName:     resourceTypeName,
		TypesInfo:            info,
		ModelFieldToTFSchema: make(map[string]string),
	}

	// Single pass: collect all information from file.Decls
	structs := make(map[string]*ast.StructType)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Collect struct definitions
			for _, spec := range d.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						structs[typeSpec.Name.Name] = structType
					}
				}
			}

		case *ast.FuncDecl:
			if d.Recv == nil || len(d.Recv.List) == 0 {
				continue
			}

			recvType := GetReceiverTypeName(d.Recv.List[0].Type)
			if recvType != resourceTypeName {
				continue
			}

			// Collect methods by name
			switch d.Name.Name {
			case "ModelObject":
				// Extract model type name from: return &ModelName{}
				if d.Body != nil {
					ast.Inspect(d.Body, func(n ast.Node) bool {
						ret, ok := n.(*ast.ReturnStmt)
						if !ok || len(ret.Results) == 0 {
							return true
						}
						// Match &ModelName{} pattern
						if unaryExpr, ok := ret.Results[0].(*ast.UnaryExpr); ok {
							if compLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
								if ident, ok := compLit.Type.(*ast.Ident); ok {
									result.ModelName = ident.Name
									return false
								}
							}
						}
						return true
					})
				}

			case "Arguments":
				result.ArgumentsFunc = d
			case "Attributes":
				result.AttributesFunc = d
			case "Create":
				result.CreateFunc = d
			case "Read":
				result.ReadFunc = d
			case "Update":
				result.UpdateFunc = d
				result.UpdateFuncBody = extractUpdateFuncBody(d)
			case "Delete":
				result.DeleteFunc = d
			}
		}
	}

	// Resolve model struct from collected structs
	if result.ModelName != "" {
		if modelStruct, ok := structs[result.ModelName]; ok {
			result.ModelStruct = modelStruct
			result.parseModelStruct(modelStruct)
		}
	}

	return result
}

// parsing model struct
func (info *TypedResourceInfo) parseModelStruct(modelStruct *ast.StructType) {
	for _, field := range modelStruct.Fields.List {
		if field.Tag == nil {
			continue
		}

		tagValue := strings.Trim(field.Tag.Value, "`")
		if !strings.Contains(tagValue, "tfschema:") {
			continue
		}

		// Extract tfschema tag value
		parts := strings.Split(tagValue, `"`)
		if len(parts) >= 2 {
			tfschemaName := parts[1]
			if len(field.Names) > 0 {
				// Map: struct field name -> tfschema name
				info.ModelFieldToTFSchema[field.Names[0].Name] = tfschemaName
			}
		}
	}
}

// GetReceiverTypeName extracts the type name from a method receiver
func GetReceiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// IsResourceWithUpdateInterface checks if a type implements sdk.ResourceWithUpdate
func IsResourceWithUpdateInterface(expr ast.Expr) bool {
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkgIdent, ok := selExpr.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "sdk" {
		return false
	}

	return selExpr.Sel.Name == "ResourceWithUpdate"
}

// extractUpdateFuncBody extracts the function body from sdk.ResourceFunc{ Func: func(...) {...} }
func extractUpdateFuncBody(updateFunc *ast.FuncDecl) *ast.BlockStmt {
	if updateFunc == nil || updateFunc.Body == nil {
		return nil
	}

	var updateFuncBody *ast.BlockStmt
	ast.Inspect(updateFunc.Body, func(n ast.Node) bool {
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

	return updateFuncBody
}

// Get schema map returned from func directly
func GetSchemaMapReturnedFromFunc(funcDecl *ast.FuncDecl) *ast.CompositeLit {
	var schemaMap *ast.CompositeLit
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		if compLit, ok := ret.Results[0].(*ast.CompositeLit); ok {
			if IsSchemaMap(compLit) {
				schemaMap = compLit
				return false
			}
		}

		return true
	})
	return schemaMap
}
