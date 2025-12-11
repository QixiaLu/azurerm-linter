package AZC005

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
)

const analyzerName = "AZC005"

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  "check that percentage properties use _percentage suffix instead of _in_percent",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip migration packages
	if strings.Contains(pass.Pkg.Path(), "/migration") {
		return nil, nil
	}

	for _, f := range pass.Files {
		filename := pass.Fset.File(f.Pos()).Name()

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Look for composite literals that represent schema maps
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if this is a map[string]*schema.Schema
			mapType, ok := comp.Type.(*ast.MapType)
			if !ok {
				return true
			}
			if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
				return true
			}
			starExpr, ok := mapType.Value.(*ast.StarExpr)
			if !ok {
				return true
			}
			selExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || selExpr.Sel.Name != "Schema" {
				return true
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
						pass.Reportf(kv.Pos(), "%s: field %q should use '_percentage' suffix instead of '_in_percent' (suggested: %q)", analyzerName, fieldName, suggestedName)
					}
				}
			}

			return true
		})
	}

	return nil, nil
}
