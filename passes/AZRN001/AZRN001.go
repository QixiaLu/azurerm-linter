package AZRN001

import (
	"go/ast"
	"strings"

	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/util"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const analyzerName = "AZRN001"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "check that percentage properties use _percentage suffix instead of _in_percent",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip migration packages
	if strings.Contains(pass.Pkg.Path(), "/migration") {
		return nil, nil
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}
	nodeFilter := []ast.Node{(*ast.CompositeLit)(nil)}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		// Apply filename filtering
		filename := pass.Fset.File(comp.Pos()).Name()
		if !changedlines.IsFileChanged(filename) || strings.HasSuffix(filename, "_test.go") {
			return
		}

		// Check if this is a map[string]*schema.Schema
		mapType, ok := comp.Type.(*ast.MapType)
		if !ok {
			return
		}
		if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
			return
		}
		starExpr, ok := mapType.Value.(*ast.StarExpr)
		if !ok {
			return
		}
		selExpr, ok := starExpr.X.(*ast.SelectorExpr)
		if !ok || selExpr.Sel.Name != "Schema" {
			return
		}

		// Iterate through the schema fields
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			// Get the field name
			keyLit, ok := kv.Key.(*ast.BasicLit)
			if !ok {
				continue
			}
			fieldName := strings.Trim(keyLit.Value, `"`)

			// Check if field name contains "_in_percent"
			if strings.Contains(fieldName, "_in_percent") {
				suggestedName := strings.ReplaceAll(fieldName, "_in_percent", "_percentage")
				pos := pass.Fset.Position(kv.Pos())
				// Only report if this line is in the changed lines (or filter is disabled)
				if changedlines.ShouldReport(pos.Filename, pos.Line) {
					pass.Reportf(kv.Pos(), "%s: field %q should use %s suffix instead of %s (suggested: %q)\n",
						analyzerName, fieldName,
						util.FixedCode("'_percentage'"),
						util.IssueLine("'_in_percent'"),
						suggestedName)
				}
			}
		}
	})

	return nil, nil
}
