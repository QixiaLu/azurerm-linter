package schemainfo

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
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

var Analyzer = &analysis.Analyzer{
	Name:       "schemainfo",
	Doc:        "extracts schema information from commonschema and other helper packages",
	Run:        run,
	ResultType: reflect.TypeOf(&SchemaInfo{}),
}

// Global cache for schema info
var (
	cachedSchemaInfo *SchemaInfo
	cacheMutex       sync.Mutex
	cacheInitialized bool
)

func run(pass *analysis.Pass) (interface{}, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Return cached result if already initialized
	if cacheInitialized {
		return cachedSchemaInfo, nil
	}

	info := &SchemaInfo{
		Functions: make(map[string]SchemaProperties),
	}

	// Directly load commonschema package (don't rely on imports)
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}

	// Load commonschema package
	pkgs, err := packages.Load(cfg, "github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema")
	if err == nil && len(pkgs) > 0 {
		parseHelperPackage(pkgs[0], info)
	}

	// Cache the result
	cachedSchemaInfo = info
	cacheInitialized = true

	return info, nil
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

// Old implementation kept for reference - now using direct load
func runOld(pass *analysis.Pass) (interface{}, error) {
	info := &SchemaInfo{
		Functions: make(map[string]SchemaProperties),
	}

	// Find commonschema and other helper packages in imports
	for _, imp := range pass.Pkg.Imports() {
		if strings.HasSuffix(imp.Path(), "commonschema") ||
			strings.Contains(imp.Path(), "helpers") {
			// Load the package with full syntax
			cfg := &packages.Config{
				Mode: packages.LoadAllSyntax,
				Fset: pass.Fset,
			}

			pkgs, err := packages.Load(cfg, imp.Path())
			if err != nil || len(pkgs) == 0 {
				continue
			}

			helperPkg := pkgs[0]
			parseHelperPackage(helperPkg, info)
		}
	}

	return info, nil
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
				compLit, _ = expr.X.(*ast.CompositeLit)
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
