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

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		filename := pass.Fset.Position(comp.Pos()).Filename
		if !loader.IsFileChanged(filename) {
			return
		}

		skipFile := false
		for _, skip := range skipFileSuffix {
			if strings.HasSuffix(filename, skip) {
				skipFile = true
				break
			}
		}
		if skipFile {
			return
		}

		// Skip if it's not a schemaMap
		// TODO: could also detect if the schema is defined as &pluginsdk.Schema{} or &schema.Schema{}
		if !helper.IsSchemaMap(comp) {
			return
		}

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
			if !ok {
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
	})

	return schemaInfoMap, nil
}
