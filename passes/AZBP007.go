package passes

import (
	"go/ast"
	"go/types"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZBP007Doc = `check that string slices are initialized using make instead of empty literals

The AZBP007 analyzer reports cases where string slices are initialized using empty
composite literals like []string{} instead of make([]string, 0).

Using make() is preferred for consistency and clarity.

Example violation:
    result := []string{}

Valid usage:
    result := make([]string, 0)
`

const azbp007Name = "AZBP007"

var AZBP007Analyzer = &analysis.Analyzer{
	Name:     azbp007Name,
	Doc:      AZBP007Doc,
	Run:      runAZBP007,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP007(pass *analysis.Pass) (interface{}, error) {
	if helper.ShouldSkipPackageForResourceAnalysis(pass.Pkg.Path()) {
		return nil, nil
	}

	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	// Check variable declarations: var x = []string{} or x := []string{}
	nodeFilter := []ast.Node{(*ast.AssignStmt)(nil), (*ast.ValueSpec)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		var rhsExprs []ast.Expr

		switch node := n.(type) {
		case *ast.AssignStmt:
			// x := []string{} or x = []string{}
			rhsExprs = node.Rhs
		case *ast.ValueSpec:
			// var x = []string{}
			rhsExprs = node.Values
		}

		for _, expr := range rhsExprs {
			compositeLit, ok := expr.(*ast.CompositeLit)
			if !ok {
				continue
			}

			// Check if it's a slice type in AST
			arrayType, ok := compositeLit.Type.(*ast.ArrayType)
			if !ok {
				continue
			}

			// Make sure it's a slice (no length specified) not an array
			if arrayType.Len != nil {
				continue
			}

			// Only flag empty slice literals
			if len(compositeLit.Elts) > 0 {
				continue
			}

			// Only check []string slices
			t := pass.TypesInfo.TypeOf(compositeLit)
			slice, ok := t.(*types.Slice)
			if !ok {
				continue
			}
			basic, ok := slice.Elem().(*types.Basic)
			if !ok || basic.Kind() != types.String {
				continue
			}

			pos := pass.Fset.Position(compositeLit.Pos())
			if !loader.ShouldReport(pos.Filename, pos.Line) {
				continue
			}

			if ignorer.ShouldIgnore(azbp007Name, compositeLit) {
				continue
			}

			pass.Reportf(compositeLit.Pos(), "%s: use %s instead of %s\n",
				azbp007Name,
				helper.FixedCode("make([]string, 0)"),
				helper.IssueLine("[]string{}"))
		}
	})

	return nil, nil
}
