package schemainfo

import (
	"go/ast"
	"go/token"
	"sync"

	"golang.org/x/tools/go/packages"
)

// SchemaInfo stores information about schema functions
type SchemaInfo struct {
	// Map of package.FunctionName -> SchemaProperties
	Functions map[string]SchemaProperties
}

type SchemaProperties struct {
	Required bool
	Optional bool
	Computed bool
}

// Global cache for schema info
var (
	cachedSchemaInfo *SchemaInfo
	once             sync.Once
)

// GetSchemaInfo returns cached schema information, loading it on first call
func GetSchemaInfo() *SchemaInfo {
	once.Do(func() {
		cachedSchemaInfo = loadSchemaInfo()
	})
	return cachedSchemaInfo
}

func loadSchemaInfo() *SchemaInfo {
	info := &SchemaInfo{
		Functions: make(map[string]SchemaProperties),
	}

	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}

	pkgs, err := packages.Load(cfg, "github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema")
	if err == nil && len(pkgs) > 0 {
		parseHelperPackage(pkgs[0], info)
	}

	return info
}

func parseHelperPackage(helperPkg *packages.Package, info *SchemaInfo) {
	// Parse all functions in the package
	for _, file := range helperPkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil {
				return true
			}

			// Only process exported functions (that return schemas)
			if !funcDecl.Name.IsExported() {
				return true
			}

			// Extract schema properties from function body
			props := extractSchemaPropertiesFromFunc(funcDecl)
			if props != nil {
				key := helperPkg.PkgPath + "." + funcDecl.Name.Name
				info.Functions[key] = *props
			}

			return true
		})
	}
}

func extractSchemaPropertiesFromFunc(funcDecl *ast.FuncDecl) *SchemaProperties {
	var props SchemaProperties

	// Look for return statements with &schema.Schema{...}
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		// Handle &schema.Schema{...}
		var compLit *ast.CompositeLit

		switch expr := ret.Results[0].(type) {
		case *ast.UnaryExpr:
			if expr.Op == token.AND {
				if cl, ok := expr.X.(*ast.CompositeLit); ok {
					compLit = cl
				}
			}
		case *ast.CompositeLit:
			compLit = expr
		}

		if compLit == nil {
			return true
		}

		// Extract Required/Optional/Computed from composite literal
		for _, elt := range compLit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}

			switch key.Name {
			case "Required":
				if val, ok := kv.Value.(*ast.Ident); ok && val.Name == "true" {
					props.Required = true
				}
			case "Optional":
				if val, ok := kv.Value.(*ast.Ident); ok && val.Name == "true" {
					props.Optional = true
				}
			case "Computed":
				if val, ok := kv.Value.(*ast.Ident); ok && val.Name == "true" {
					props.Computed = true
				}
			}
		}

		return false
	})

	return &props
}
