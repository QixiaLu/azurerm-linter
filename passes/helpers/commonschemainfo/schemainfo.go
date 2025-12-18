package commonschemainfo

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// SchemaInfo stores information about common schema functions
type SchemaInfo struct {
	// Map of package.FunctionName -> *schema.SchemaInfo
	Functions map[string]*schema.SchemaInfo
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
	return &SchemaInfo{Functions: make(map[string]*schema.SchemaInfo)}, nil
}

func loadSchemaInfo(pass *analysis.Pass) *SchemaInfo {
	info := &SchemaInfo{
		Functions: make(map[string]*schema.SchemaInfo),
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

			// Extract schema info from function body using package's TypesInfo
			schemaInfo := extractSchemaPropertiesFromFunc(funcDecl, helperPkg.TypesInfo)
			if schemaInfo != nil {
				key := helperPkg.PkgPath + "." + funcDecl.Name.Name
				info.Functions[key] = schemaInfo
			}

			return true
		})
	}
}

func extractSchemaPropertiesFromFunc(funcDecl *ast.FuncDecl, typesInfo *types.Info) *schema.SchemaInfo {
	// Look for return statements with &schema.Schema{...}
	var returnedSchema *ast.CompositeLit

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		// Handle &schema.Schema{...}
		var compLit *ast.CompositeLit

		switch expr := ret.Results[0].(type) {
		case *ast.UnaryExpr:
			// Handle &schema.Schema{...}
			if cl, ok := expr.X.(*ast.CompositeLit); ok {
				compLit = cl
			}
		case *ast.CompositeLit:
			compLit = expr
		}

		if compLit != nil {
			returnedSchema = compLit
			return false // Stop inspection
		}

		return true
	})

	if returnedSchema == nil {
		return nil
	}

	// Parse the returned schema using tfproviderlint's NewSchemaInfo with the package's TypesInfo
	return schema.NewSchemaInfo(returnedSchema, typesInfo)
}
