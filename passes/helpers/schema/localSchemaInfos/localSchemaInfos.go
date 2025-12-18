package schema

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemafields"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `Gather all inline schema infos declared in the package`

const analyzerName = "localSchemaInfos"

type SchemaInfoWithName struct {
	Info         *schema.SchemaInfo
	PropertyName string
}

var Analyzer = &analysis.Analyzer{
	Name:       analyzerName,
	Doc:        Doc,
	Run:        run,
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeOf(map[*ast.CompositeLit]*SchemaInfoWithName{}),
}

var skipPackages = []string{"_test", "/migration", "/client", "/validate", "/test-data", "/parse", "/models"}
var skipFileSuffix = []string{"_test.go", "registration.go"}

func run(pass *analysis.Pass) (interface{}, error) {
	schemaInfoMap := make(map[*ast.CompositeLit]*SchemaInfoWithName)

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
		if !changedlines.IsFileChanged(filename) {
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
		if !schemafields.IsSchemaMap(comp) {
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
				schemaInfoMap[schemaLit] = &SchemaInfoWithName{
					Info:         schemaInfo,
					PropertyName: propertyName,
				}
			}
		}
	})

	return schemaInfoMap, nil
}
