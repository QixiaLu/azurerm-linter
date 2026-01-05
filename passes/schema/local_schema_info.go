package schema

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const localAnalyzerName = "localschemainfo"

type LocalSchemaInfoWithName struct {
	Info         *schema.SchemaInfo
	PropertyName string
}

var LocalAnalyzer = &analysis.Analyzer{
	Name:       localAnalyzerName,
	Doc:        "Gather all inline schema infos declared in the package",
	Run:        runLocal,
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeOf(map[*ast.CompositeLit]*LocalSchemaInfoWithName{}),
}

var skipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var skipFileSuffix = []string{"_test.go", "registration.go"}

func runLocal(pass *analysis.Pass) (interface{}, error) {
	schemaInfoMap := make(map[*ast.CompositeLit]*LocalSchemaInfoWithName)

	pkgPath := pass.Pkg.Path()
	for _, skip := range skipPackages {
		if strings.Contains(pkgPath, skip) {
			return schemaInfoMap, nil
		}
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return schemaInfoMap, nil
	}

	skipSchemas := make(map[*ast.CompositeLit]bool)
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Skip schemas inside if statement blocks (feature flags)
			if ifStmt, ok := n.(*ast.IfStmt); ok {
				ast.Inspect(ifStmt.Body, func(node ast.Node) bool {
					if comp, ok := node.(*ast.CompositeLit); ok {
						skipSchemas[comp] = true
					}
					return true
				})
				return true
			}

			// Skip only direct Schema composite literals in Elem field values
			// e.g. Elem: &pluginsdk.Schema{Type: TypeString}
			// But do NOT skip Resource types: Elem: &pluginsdk.Resource{Schema: map[...]}
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Elem" {
					// Check if the value is a UnaryExpr with & (address-of operator)
					if unary, ok := kv.Value.(*ast.UnaryExpr); ok && unary.Op.String() == "&" {
						if comp, ok := unary.X.(*ast.CompositeLit); ok {
							// Only skip if it's a Schema type, not Resource type
							if helper.IsSchemaSchema(pass, comp) {
								skipSchemas[comp] = true
							}
						}
					}
				}
			}
			return true
		})
	}

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		filename := pass.Fset.Position(comp.Pos()).Filename
		if !shouldProcessFile(filename) {
			return
		}

		// Phase 1: Process map[string]*Schema composite literals
		if helper.IsSchemaMap(comp) {
			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				key, ok := kv.Key.(*ast.BasicLit)
				if !ok {
					continue
				}
				propertyName := strings.Trim(key.Value, `"`)

				schemaLit, ok := kv.Value.(*ast.CompositeLit)
				if !ok || skipSchemas[schemaLit] {
					continue
				}

				schemaInfo := schema.NewSchemaInfo(schemaLit, pass.TypesInfo)
				if schemaInfo != nil {
					schemaInfoMap[schemaLit] = &LocalSchemaInfoWithName{
						Info:         schemaInfo,
						PropertyName: propertyName,
					}
				}
			}
			return
		}

		// Phase 2: Process standalone &Schema composite literals
		if helper.IsSchemaSchema(pass, comp) {
			// Skip schemas inside if blocks or Elem values
			if skipSchemas[comp] {
				return
			}

			schemaInfo := schema.NewSchemaInfo(comp, pass.TypesInfo)
			if schemaInfo != nil {
				schemaInfoMap[comp] = &LocalSchemaInfoWithName{
					Info:         schemaInfo,
					PropertyName: "",
				}
			}
		}
	})

	return schemaInfoMap, nil
}

// shouldProcessFile checks if a file should be processed based on filters
func shouldProcessFile(filename string) bool {
	if !loader.IsFileChanged(filename) {
		return false
	}

	for _, skip := range skipFileSuffix {
		if strings.HasSuffix(filename, skip) {
			return false
		}
	}

	return true
}
