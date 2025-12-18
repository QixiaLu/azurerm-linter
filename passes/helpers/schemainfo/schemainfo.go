package schemainfo

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
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

// Global cache for schema info - loaded only once successfully
var (
	globalSchemaInfo *SchemaInfo
	loadOnce         sync.Once
	loadMutex        sync.RWMutex
)

func run(pass *analysis.Pass) (interface{}, error) {
	loadMutex.RLock()
	if globalSchemaInfo != nil && len(globalSchemaInfo.Functions) > 0 {
		loadMutex.RUnlock()
		return globalSchemaInfo, nil
	}
	loadMutex.RUnlock()

	loadOnce.Do(func() {
		loadMutex.Lock()
		defer loadMutex.Unlock()
		info := loadSchemaInfo(pass)
		if len(info.Functions) > 0 {
			globalSchemaInfo = info
		}
	})

	loadMutex.RLock()
	defer loadMutex.RUnlock()
	if globalSchemaInfo != nil {
		return globalSchemaInfo, nil
	}

	// Return empty info if load failed
	return &SchemaInfo{Functions: make(map[string]SchemaProperties)}, nil
}

func loadSchemaInfo(pass *analysis.Pass) *SchemaInfo {
	info := &SchemaInfo{
		Functions: make(map[string]SchemaProperties),
	}

	if len(pass.Files) == 0 {
		return info
	}

	// Get the file path from the first file in the package
	filePath := pass.Fset.Position(pass.Files[0].Pos()).Filename
	if strings.Contains(filePath, "go-build") || strings.Contains(filePath, "AppData") {
		return info
	}

	// Traverse up to find the directory containing "internal"
	dir := filepath.Dir(filePath)
	foundInternal := false
	for dir != "" && dir != "." && dir != string(filepath.Separator) {
		base := filepath.Base(dir)
		if base == "internal" {
			// Go up one more level to get the repo root
			dir = filepath.Dir(dir)
			foundInternal = true
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return info
		}
		dir = parent
	}

	if !foundInternal {
		return info
	}

	vendorPath := filepath.Join(dir, "vendor", "github.com", "hashicorp", "go-azure-helpers", "resourcemanager", "commonschema")
	if _, err := os.Stat(vendorPath); os.IsNotExist(err) {
		return info
	}

	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  vendorPath,
	}

	// Load commonschema package from vendor
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Printf("[schemainfo] Error loading package: %v\n", err)
	} else {
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
