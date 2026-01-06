package schema

import (
	"go/ast"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/bflad/tfproviderlint/helper/astutils"
	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/helper"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

const commonAnalyzerDoc = `Extracts and caches schema information from the commonschema package in vendor directory.

Key Features:
 1. Extracts schema definitions from commonschema functions (e.g., ResourceGroupName())
 2. Uses ExtractSchemaInfoFromMap to parse resource schema maps
 3. Supports commonschema cross-package calls and same-package function call resolution

Example:

	// Function in commonschema package
	func ResourceGroupName() *pluginsdk.Schema {
	    return &pluginsdk.Schema{
	        Type:     pluginsdk.TypeString,
	        Required: true,
	        ForceNew: true,
	    }
	}

	// Using commonschema in a resource
	func (r MyResource) Arguments() map[string]*pluginsdk.Schema {
	    return map[string]*pluginsdk.Schema{
	        "name": {Type: pluginsdk.TypeString, Required: true},
	        "resource_group_name": commonschema.ResourceGroupName(),  // commonschema cross-package call
	        "metadata": metadataSchema(),                             // same-package call
	    }
	}

	// Extract all fields using ExtractSchemaInfoFromMap
	fields := ExtractSchemaInfoFromMap(pass, schemaMapLiteral, commonInfo)
	// Returns: []SchemaFieldInfo{
	//   {Name: "name", SchemaInfo: {Type: String, Required: true}},
	//   {Name: "resource_group_name", SchemaInfo: {Type: String, Required: true, ForceNew: true}},
	//   {Name: "metadata", SchemaInfo: {...}},
	// }
`

// CommonSchemaInfo stores information about common schema functions.
type CommonSchemaInfo struct {
	// Map of package.FunctionName -> *schema.SchemaInfo
	Functions map[string]*schema.SchemaInfo
}

var CommonAnalyzer = &analysis.Analyzer{
	Name:       "commonschemainfo",
	Doc:        commonAnalyzerDoc,
	Run:        runCommon,
	ResultType: reflect.TypeOf(&CommonSchemaInfo{}),
}

// Global cache for schema info - loaded only once successfully
var (
	globalSchemaInfo *CommonSchemaInfo
	loadMutex        sync.RWMutex
)

func runCommon(pass *analysis.Pass) (interface{}, error) {
	loadMutex.RLock()
	cached := globalSchemaInfo
	loadMutex.RUnlock()

	if cached != nil {
		return cached, nil
	}

	loadMutex.Lock()
	defer loadMutex.Unlock()

	// Double-check: another goroutine might have loaded while we were waiting
	if globalSchemaInfo != nil {
		return globalSchemaInfo, nil
	}

	info := loadSchemaInfo(pass)

	if len(info.Functions) > 0 {
		globalSchemaInfo = info
		return info, nil
	} else {
		// Failure: don't cache, allow retry on next call
		return info, nil
	}
}

func loadSchemaInfo(pass *analysis.Pass) *CommonSchemaInfo {
	info := &CommonSchemaInfo{
		Functions: make(map[string]*schema.SchemaInfo),
	}

	if len(pass.Files) == 0 {
		return info
	}

	// Get the file path from the first file in the package
	filePath := pass.Fset.Position(pass.Files[0].Pos()).Filename
	// These are go local cache files
	if strings.Contains(filePath, "go-build") || strings.Contains(filePath, "AppData") || strings.Contains(filePath, ".test") {
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
		log.Printf("Warning: failed to load commonschema package: %v", err)
	} else if len(pkgs) > 0 {
		parseHelperPackage(pkgs[0], info)
	}

	return info
}

func parseHelperPackage(helperPkg *packages.Package, info *CommonSchemaInfo) {
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
			schemaInfo := extractSingleSchemaPropertyFromFunc(funcDecl, helperPkg.TypesInfo)
			if schemaInfo != nil {
				key := helperPkg.PkgPath + "." + funcDecl.Name.Name
				info.Functions[key] = schemaInfo
			}

			return true
		})
	}
}

// Extract schema info from &schema.Schema{...} or &pluginsdk.Schema{} or variable declared inside the function
func extractSingleSchemaPropertyFromFunc(funcDecl *ast.FuncDecl, typesInfo *types.Info) *schema.SchemaInfo {
	var returnedSchema *ast.CompositeLit

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		switch expr := ret.Results[0].(type) {
		case *ast.UnaryExpr:
			// Handle &schema.Schema{...}
			if cl, ok := expr.X.(*ast.CompositeLit); ok {
				returnedSchema = cl
				return false
			}
		case *ast.CompositeLit:
			// Handle schema.Schema{...}
			returnedSchema = expr
			return false
		case *ast.Ident:
			// Handle return schema (variable reference)
			if compLit := helper.TraceIdentToCompositeLit(typesInfo, expr, funcDecl); compLit != nil {
				if helper.IsSchemaSchema(typesInfo, compLit) {
					returnedSchema = compLit
				}
			}
			return false
		}

		return true
	})

	if returnedSchema != nil && helper.IsSchemaSchema(typesInfo, returnedSchema) {
		return schema.NewSchemaInfo(returnedSchema, typesInfo)
	}

	return nil
}

// ExtractSchemaInfoFromMap extracts schema fields from a map[string]*schema.Schema composite literal
func ExtractSchemaInfoFromMap(pass *analysis.Pass, smap *ast.CompositeLit, commonSchemaInfo *CommonSchemaInfo) []helper.SchemaFieldInfo {
	fields := make([]helper.SchemaFieldInfo, 0, len(smap.Elts))

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

		fields = append(fields, helper.SchemaFieldInfo{
			Name:       *fieldName,
			SchemaInfo: resolvedSchema,
			Position:   i,
		})
	}

	return fields
}

// resolveSchemaInfoFromCall resolves schema info from a function call
// It tries cross-package cache first, then same-package resolution
func resolveSchemaInfoFromCall(pass *analysis.Pass, call *ast.CallExpr, commonSchemaInfo *CommonSchemaInfo) *schema.SchemaInfo {
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
	// TODO: Add search in all pkgs?
	funcDecl := helper.FindFuncDecl(pass, funcObj)
	if funcDecl == nil {
		return nil
	}

	return extractSingleSchemaPropertyFromFunc(funcDecl, pass.TypesInfo)
}
