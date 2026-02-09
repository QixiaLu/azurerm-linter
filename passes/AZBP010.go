package passes

import (
	"go/ast"
	"go/token"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZBP010Doc = `check for variables that are declared and immediately returned

The AZBP010 analyzer reports when a variable is declared and then immediately returned
on the next statement without any other usage, which could be simplified by returning
the value directly.

Example violations:

	func badExample() string {
		result := "hello"  // Declared here
		return result      // Immediately returned
	}

	func badConstant() int {
		const num = 42
		return num
	}

Correct usage:

	func goodExample() string {
		return "hello"  // Return value directly
	}

	func goodUsage() string {
		result := "hello"
		fmt.Println("Processing:", result)  // Variable is used
		return result
	}
`

const azbp010Name = "AZBP010"

var AZBP010Analyzer = &analysis.Analyzer{
	Name:     azbp010Name,
	Doc:      AZBP010Doc,
	Run:      runAZBP010,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP010(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}
		if funcDecl.Body == nil {
			return
		}

		statements := funcDecl.Body.List
		if len(statements) < 2 {
			return
		}

		// Check consecutive pairs of statements
		for i := 0; i < len(statements)-1; i++ {
			declStmt := statements[i]
			returnStmt, ok := statements[i+1].(*ast.ReturnStmt)
			if !ok {
				continue
			}

			declaredVars := getVariableDeclarations(declStmt)
			if len(declaredVars) == 0 {
				continue
			}

			if returnsOnlyDeclaredVars(returnStmt, declaredVars) {
				pos := pass.Fset.Position(declStmt.Pos())
				if !loader.ShouldReport(pos.Filename, pos.Line) {
					continue
				}
				if ignorer.ShouldIgnore(azbp010Name, declStmt) {
					continue
				}

				varNames := make([]string, len(declaredVars))
				copy(varNames, declaredVars)

				pass.Reportf(declStmt.Pos(), "%s: variable declared and immediately returned, consider returning the value directly\n",
					azbp010Name)
			}
		}
	})

	return nil, nil
}

// getVariableDeclarations extracts variable names from declaration statements
func getVariableDeclarations(stmt ast.Stmt) []string {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		if genDecl, ok := s.Decl.(*ast.GenDecl); ok && (genDecl.Tok == token.VAR || genDecl.Tok == token.CONST) {
			var vars []string
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						vars = append(vars, name.Name)
					}
				}
			}
			return vars
		}
	case *ast.AssignStmt:
		if s.Tok == token.DEFINE {
			var vars []string
			for _, lhs := range s.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					vars = append(vars, ident.Name)
				}
			}
			return vars
		}
	}
	return nil
}

// returnsOnlyDeclaredVars checks if return statement returns exactly the declared variables
func returnsOnlyDeclaredVars(returnStmt *ast.ReturnStmt, declaredVars []string) bool {
	if len(returnStmt.Results) != len(declaredVars) {
		return false
	}

	// Create a map for quick lookup
	declaredMap := make(map[string]bool)
	for _, varName := range declaredVars {
		declaredMap[varName] = true
	}

	// Check if all returned expressions are exactly the declared variables
	for _, result := range returnStmt.Results {
		if ident, ok := result.(*ast.Ident); ok {
			if !declaredMap[ident.Name] {
				return false
			}
		} else {
			// Return statement contains non-identifier (e.g., function call, literal)
			return false
		}
	}

	return true
}
