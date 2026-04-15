package passes

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZBP013Doc = `check for chained nil checks that should be split into separate if statements

The AZBP013 analyzer reports when an if statement uses || to combine multiple
nil checks in a chain (where one checked expression is a prefix of the next)
and the body returns an error. Each nil condition should be a separate if
statement so the error message can identify exactly which value was nil.

Example violation:

  if resp.Model == nil || resp.Model.Properties == nil {
      return fmt.Errorf("retrieving %s: model was nil", id)
  }

Valid usage:

  if resp.Model == nil {
      return fmt.Errorf("retrieving %s: model was nil", id)
  }
  if resp.Model.Properties == nil {
      return fmt.Errorf("retrieving %s: properties was nil", id)
  }
`

const azbp013Name = "AZBP013"

var AZBP013Analyzer = &analysis.Analyzer{
	Name:     azbp013Name,
	Doc:      AZBP013Doc,
	Run:      runAZBP013,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP013(pass *analysis.Pass) (interface{}, error) {
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

	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return
		}

		pos := pass.Fset.Position(ifStmt.Pos())
		if !loader.IsFileChanged(pos.Filename) {
			return
		}
		if ignorer.ShouldIgnore(azbp013Name, ifStmt) {
			return
		}

		if ifStmt.Else != nil {
			return
		}

		nilExprs := collectOrNilChecks(ifStmt.Cond, pass)
		if len(nilExprs) < 2 {
			return
		}

		if !hasChainedPrefix(nilExprs) {
			return
		}

		if !bodyReturnsError(ifStmt.Body, pass) {
			return
		}

		reporting.Reportf(pass, reporting.ReportOptions{
			Rule:          azbp013Name,
			ReportPos:     ifStmt.Pos(),
			EvidenceFile:  pos.Filename,
			EvidenceLines: []int{pos.Line},
			MatchMode:     reporting.MatchModeExactAdded,
		}, "%s: split chained nil checks into separate if statements with distinct error messages\n",
			azbp013Name)
	})

	return nil, nil
}

// collectOrNilChecks walks || chains and collects the target of each "expr == nil".
// Returns nil if any operand is not a nil comparison.
func collectOrNilChecks(expr ast.Expr, pass *analysis.Pass) []string {
	binExpr, ok := expr.(*ast.BinaryExpr)
	if !ok || binExpr.Op.String() != "||" {
		s := extractNilCompareTarget(expr, pass)
		if s == "" {
			return nil
		}
		return []string{s}
	}

	left := collectOrNilChecks(binExpr.X, pass)
	if left == nil {
		return nil
	}
	right := collectOrNilChecks(binExpr.Y, pass)
	if right == nil {
		return nil
	}
	return append(left, right...)
}

// extractNilCompareTarget returns the non-nil side of "expr == nil", or "".
func extractNilCompareTarget(expr ast.Expr, pass *analysis.Pass) string {
	binExpr, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return ""
	}
	if binExpr.Op.String() != "==" {
		return ""
	}

	if isNilExpr(binExpr.Y, pass) {
		return types.ExprString(binExpr.X)
	}
	if isNilExpr(binExpr.X, pass) {
		return types.ExprString(binExpr.Y)
	}
	return ""
}

func isNilExpr(expr ast.Expr, pass *analysis.Pass) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.Uses[ident]
	_, isNil := obj.(*types.Nil)
	return isNil
}

// hasChainedPrefix checks that at least one consecutive pair has a dot-prefix
// relationship, e.g. "resp.Model" is a prefix of "resp.Model.Properties".
func hasChainedPrefix(exprs []string) bool {
	for i := 0; i < len(exprs)-1; i++ {
		if isExprPrefix(exprs[i], exprs[i+1]) {
			return true
		}
	}
	return false
}

func isExprPrefix(a, b string) bool {
	if !strings.HasPrefix(b, a) {
		return false
	}
	rest := b[len(a):]
	return len(rest) > 0 && rest[0] == '.'
}

// bodyReturnsError checks if the body contains a return with fmt.Errorf or errors.New.
func bodyReturnsError(body *ast.BlockStmt, _ *analysis.Pass) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		retStmt, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, result := range retStmt.Results {
			if isErrorCall(result) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func isErrorCall(expr ast.Expr) bool {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkg := ident.Name
	fn := sel.Sel.Name
	return (pkg == "fmt" && fn == "Errorf") || (pkg == "errors" && fn == "New")
}
