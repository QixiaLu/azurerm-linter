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

const AZBP012Doc = `check for unnecessary else blocks that can be avoided by setting a default

The AZBP012 analyzer reports when an if/else block assigns a value to the same
target in both branches, and the else branch could be hoisted as a default
assignment before the if statement.

Example violation:
  if len(regions) != 0 {
      props.Type = pointer.To(TypeManaged)
  } else {
      props.Type = pointer.To(TypeUnmanaged)
  }

Valid usage:
  props.Type = pointer.To(TypeUnmanaged)
  if len(regions) != 0 {
      props.Type = pointer.To(TypeManaged)
  }
`

const azbp012Name = "AZBP012"

var AZBP012Analyzer = &analysis.Analyzer{
	Name:     azbp012Name,
	Doc:      AZBP012Doc,
	Run:      runAZBP012,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP012(pass *analysis.Pass) (interface{}, error) {
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

	// Collect all IfStmts that appear as the Else branch of another IfStmt
	// so we can skip them (they are part of an else-if chain).
	elseIfs := map[*ast.IfStmt]bool{}
	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)
		if inner, ok := ifStmt.Else.(*ast.IfStmt); ok {
			elseIfs[inner] = true
		}
	})

	insp.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)

		if elseIfs[ifStmt] {
			return
		}

		pos := pass.Fset.Position(ifStmt.Pos())
		if !loader.ShouldReport(pos.Filename, pos.Line) {
			return
		}
		if ignorer.ShouldIgnore(azbp012Name, ifStmt) {
			return
		}

		if target := simpleIfElseSameTarget(ifStmt); target != "" {
			pass.Reportf(ifStmt.Pos(),
				"%s: simplify if/else assigning `%s` by setting the else value as the default before the if\n",
				azbp012Name, target)
		}
	})

	return nil, nil
}

// simpleIfElseSameTarget returns the assignment target if:
//   - The else branch is a plain block (not else-if)
//   - Each branch contains exactly one statement
//   - Both statements are assignments to the same target
//
// Returns "" if the pattern does not match.
func simpleIfElseSameTarget(ifStmt *ast.IfStmt) string {
	// Must have an else branch
	elseBlock, ok := ifStmt.Else.(*ast.BlockStmt)
	if !ok {
		return ""
	}

	// Each branch must contain exactly one statement
	if len(ifStmt.Body.List) != 1 || len(elseBlock.List) != 1 {
		return ""
	}

	ifAssign := singleAssignTarget(ifStmt.Body.List[0])
	elseAssign := singleAssignTarget(elseBlock.List[0])
	if ifAssign == "" || elseAssign == "" {
		return ""
	}

	if ifAssign != elseAssign {
		return ""
	}
	return ifAssign
}

// singleAssignTarget returns the string representation of the LHS if the
// statement is a simple assignment (=) with exactly one target, or "" otherwise.
func singleAssignTarget(stmt ast.Stmt) string {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok {
		return ""
	}
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return ""
	}
	return types.ExprString(assign.Lhs[0])
}
