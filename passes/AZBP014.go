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

const AZBP014Doc = `check for empty OperationOptions literals when a Default* constructor exists

The AZBP014 analyzer reports when code uses an empty struct literal like
SomeOperationOptions{} when the package provides a DefaultSomeOperationOptions()
constructor. The Default function should be used for forward compatibility.

Example violation:

  options := services.GetOperationOptions{}

Valid usage:

  options := services.DefaultGetOperationOptions()
`

const azbp014Name = "AZBP014"

var AZBP014Analyzer = &analysis.Analyzer{
	Name:     azbp014Name,
	Doc:      AZBP014Doc,
	Run:      runAZBP014,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP014(pass *analysis.Pass) (interface{}, error) {
	if helper.ShouldSkipPackageForResourceAnalysis(pass.Pkg.Path()) {
		return nil, nil
	}

	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.CompositeLit)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		lit := n.(*ast.CompositeLit)

		// Only empty literals (no field initializers)
		if len(lit.Elts) != 0 {
			return
		}

		pos := pass.Fset.Position(lit.Pos())
		if !loader.ShouldReport(pos.Filename, pos.Line) {
			return
		}
		if ignorer.ShouldIgnore(azbp014Name, lit) {
			return
		}

		// Resolve the type of the composite literal
		litType := pass.TypesInfo.TypeOf(lit)
		if litType == nil {
			return
		}

		named, ok := litType.(*types.Named)
		if !ok {
			return
		}

		typeName := named.Obj().Name()
		pkg := named.Obj().Pkg()
		if pkg == nil {
			return
		}

		// Check if Default<TypeName>() exists in the type's package
		defaultFuncName := "Default" + typeName
		obj := pkg.Scope().Lookup(defaultFuncName)
		if obj == nil {
			return
		}

		// Verify it's a function
		if _, ok := obj.(*types.Func); !ok {
			return
		}

		pass.Reportf(lit.Pos(),
			"%s: use %s.%s() instead of empty %s literal\n",
			azbp014Name, pkg.Name(), defaultFuncName, typeName)
	})

	return nil, nil
}
