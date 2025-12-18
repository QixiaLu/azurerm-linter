package AZRN001

import (
	"go/ast"
	"strings"

	"github.com/qixialu/azurerm-linter/passes/changedlines"
	localschema "github.com/qixialu/azurerm-linter/passes/helpers/schema/localSchemaInfos"
	"github.com/qixialu/azurerm-linter/passes/util"
	"golang.org/x/tools/go/analysis"
)

const analyzerName = "AZRN001"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "check that percentage properties use _percentage suffix instead of _in_percent",
	Run:      run,
	Requires: []*analysis.Analyzer{localschema.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	schemaInfoCache, ok := pass.ResultOf[localschema.Analyzer].(map[*ast.CompositeLit]*localschema.SchemaInfoWithName)
	if !ok {
		return nil, nil
	}

	for schemaLit, cached := range schemaInfoCache {
		fieldName := cached.PropertyName

		// Check if field name contains "_in_percent"
		if strings.Contains(fieldName, "_in_percent") {
			suggestedName := strings.ReplaceAll(fieldName, "_in_percent", "_percentage")
			pos := pass.Fset.Position(schemaLit.Pos())
			// Only report if this line is in the changed lines
			if changedlines.ShouldReport(pos.Filename, pos.Line) {
				pass.Reportf(schemaLit.Pos(), "%s: field %q should use %s suffix instead of %s (suggested: %q)\n",
					analyzerName, fieldName,
					util.FixedCode("'_percentage'"),
					util.IssueLine("'_in_percent'"),
					suggestedName)
			}
		}
	}

	return nil, nil
}
