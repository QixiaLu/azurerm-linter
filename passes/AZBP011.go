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

const AZBP011Doc = `check for unnecessary string casting in enum comparisons

The AZBP011 analyzer reports when code uses strings.EqualFold with string type casting
on enum values that could be compared directly. This promotes type safety and better performance.

Example violations:

	// Bad - unnecessary string casting and case-insensitive comparison
	if strings.EqualFold(string(pointer.From(hibernateSupport)), string(devboxdefinitions.HibernateSupportDisabled)) {
		// ...
	}

	// Bad - both sides are enum values cast to strings
	result := strings.EqualFold(string(enumValue1), string(enumValue2))

Correct usage:

	// Good - direct enum comparison
	if pointer.From(hibernateSupport) == devboxdefinitions.HibernateSupportDisabled {
		// ...
	}

	// Good - direct enum comparison
	result := enumValue1 == enumValue2

Legitimate use cases (not flagged):

	// OK - comparing user input with enum
	if strings.EqualFold(userInput, string(enumValue)) {
		// ...
	}

	// OK - API workaround with explanation
	if strings.EqualFold(apiResponse, string(enumValue)) { //nolint:AZBP011 // API returns inconsistent casing
		// ...
	}
`

const azbp011Name = "AZBP011"

var AZBP011Analyzer = &analysis.Analyzer{
	Name:     azbp011Name,
	Doc:      AZBP011Doc,
	Run:      runAZBP011,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP011(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	// Look for strings.EqualFold calls
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Check if this is a strings.EqualFold call
		if !isStringsEqualFoldCall(callExpr) {
			return
		}

		if len(callExpr.Args) != 2 {
			return
		}

		arg1 := callExpr.Args[0]
		arg2 := callExpr.Args[1]

		if !isStringTypeCast(arg1) || !isStringTypeCast(arg2) {
			return
		}

		if !isAzureSDKEnumCast(pass, arg1) || !isAzureSDKEnumCast(pass, arg2) {
			return
		}

		pos := pass.Fset.Position(callExpr.Pos())
		if !loader.ShouldReport(pos.Filename, pos.Line) || ignorer.ShouldIgnore(azbp011Name, callExpr) {
			return
		}

		pass.Reportf(callExpr.Pos(), "%s: avoid unnecessary string casting in enum comparison, use direct enum comparison instead\n",
			azbp011Name)
	})

	return nil, nil
}

// isStringsEqualFoldCall checks if the call expression is strings.EqualFold
func isStringsEqualFoldCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "strings" && selExpr.Sel.Name == "EqualFold"
		}
	}
	return false
}

// isStringTypeCast checks if the expression is a string type cast: string(...)
func isStringTypeCast(expr ast.Expr) bool {
	if callExpr, ok := expr.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			return ident.Name == "string" && len(callExpr.Args) == 1
		}
	}
	return false
}

// isAzureSDKEnumCast checks if the string cast involves an Azure SDK enum type
func isAzureSDKEnumCast(pass *analysis.Pass, expr ast.Expr) bool {
	// Extract the inner expression from string(innerExpr)
	if callExpr, ok := expr.(*ast.CallExpr); ok && len(callExpr.Args) == 1 {
		innerExpr := callExpr.Args[0]

		if exprType := pass.TypesInfo.TypeOf(innerExpr); exprType != nil {
			if named, ok := exprType.(*types.Named); ok {
				return helper.IsAzureSDKEnumType(pass, named)
			}
		}
	}
	return false
}
