package passes

import (
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR006Doc = `check that nil checks should be performed inside flatten methods

The AZNR006 analyzer reports when code performs nil checks before calling flatten methods
instead of handling nil checks within the flatten method itself. This promotes cleaner code
and better separation of concerns.

Example violations:

	// Bad - nil check before calling flatten method (with dereferencing)
	if cloneProps.CustomerContacts != nil {
		state.CustomerContacts = flattenCloneCustomerContacts(*cloneProps.CustomerContacts)
	}

	// Bad - nil check before calling flatten method (without dereferencing)
	if cloneProps.CustomerContacts != nil {
		state.CustomerContacts = flattenCustomerContacts(cloneProps.CustomerContacts)
	}

Correct usage:

	// Good - flatten method handles nil checks internally
	state.CustomerContacts = flattenCloneCustomerContacts(cloneProps.CustomerContacts)

	// Inside flatten method:
	func flattenCustomerContacts(contacts *SomeType) []interface{} {
		if contacts == nil {
			return []interface{}{}
		}
		// ... flatten logic
	}
`

const aznr006Name = "AZNR006"

var AZNR006Analyzer = &analysis.Analyzer{
	Name:     aznr006Name,
	Doc:      AZNR006Doc,
	Run:      runAZNR006,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZNR006(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}
	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	// Look for if statements
	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)

		// Check if this is a nil check pattern and extract the checked variable
		checkedVar := getNilCheckedVariable(ifStmt.Cond)
		if checkedVar == nil {
			return
		}

		// Check if the body contains a single flatten method call using the checked variable
		if !containsSingleFlattenCallWithVariable(ifStmt.Body, checkedVar) {
			return
		}

		pos := pass.Fset.Position(ifStmt.Pos())
		if !loader.ShouldReport(pos.Filename, pos.Line) {
			return
		}
		if ignorer.ShouldIgnore(aznr006Name, ifStmt) {
			return
		}

		pass.Reportf(ifStmt.Pos(), "%s: perform nil checks inside the flatten method instead of before calling it\n",
			aznr006Name)
	})

	return nil, nil
}

// getNilCheckedVariable extracts the variable being nil-checked from a condition (x != nil)
// Returns the expression being checked, or nil if it's not a nil check
func getNilCheckedVariable(cond ast.Expr) ast.Expr {
	binExpr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	if binExpr.Op != token.NEQ {
		return nil
	}

	ident, ok := binExpr.Y.(*ast.Ident)
	if !ok || ident.Name != "nil" {
		return nil
	}

	return binExpr.X
}

// containsSingleFlattenCallWithVariable checks if the block contains exactly one statement
func containsSingleFlattenCallWithVariable(block *ast.BlockStmt, checkedVar ast.Expr) bool {
	// Must have exactly one statement
	if len(block.List) != 1 {
		return false
	}

	stmt := block.List[0]
	assignStmt, ok := stmt.(*ast.AssignStmt)
	if !ok {
		return false
	}

	for _, rhs := range assignStmt.Rhs {
		if callExpr, ok := rhs.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "flatten") {
				// Check if any argument uses our checked variable
				if callUsesVariable(callExpr, checkedVar) {
					return true
				}
			}
		}
	}
	return false
}

// callUsesVariable checks if any argument in the call uses the specified variable
func callUsesVariable(callExpr *ast.CallExpr, variable ast.Expr) bool {
	for _, arg := range callExpr.Args {
		// Check if argument is the variable directly
		if areExpressionsEqual(arg, variable) {
			return true
		}
		// Check if argument is the dereferenced variable (*variable)
		if starExpr, ok := arg.(*ast.StarExpr); ok {
			if areExpressionsEqual(starExpr.X, variable) {
				return true
			}
		}
	}
	return false
}

// areExpressionsEqual compares two AST expressions for structural equality
func areExpressionsEqual(x, y ast.Expr) bool {
	if x == nil || y == nil {
		return x == y
	}

	var xBuf, yBuf strings.Builder
	if err := format.Node(&xBuf, token.NewFileSet(), x); err != nil {
		return false
	}
	if err := format.Node(&yBuf, token.NewFileSet(), y); err != nil {
		return false
	}

	return xBuf.String() == yBuf.String()
}
