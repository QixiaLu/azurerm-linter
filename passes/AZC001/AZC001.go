package AZC001

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"golang.org/x/tools/go/analysis"
)

const Doc = `check for fixed error strings using fmt.Errorf instead of errors.New

The AZC001 analyzer reports cases where fixed error strings (without format placeholders)
use fmt.Errorf() instead of errors.New().

Example violations:
  fmt.Errorf("something went wrong")  // should use errors.New()
  
Valid usage:
  errors.New("something went wrong")
  fmt.Errorf("value %s is invalid", value)  // has placeholder, OK`

const analyzerName = "AZC001"

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		commentignore.Analyzer,
	},
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	ignorer := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)

	for _, f := range pass.Files {
		filePos := pass.Fset.Position(f.Pos())
		filename := filePos.Filename

		// Only check resource and data source files 
		// TODO: should be all files in services/, since some of those contain schema as well, e.g. helper.go
		if !strings.HasSuffix(filename, "_resource.go") && !strings.HasSuffix(filename, "_data_source.go") {
			continue
		}

		// Skip test files
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if we should ignore this node
			if ignorer.ShouldIgnore(analyzerName, call) {
				return true
			}

			// Check if it's a selector expression (pkg.Function)
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Check if it's calling Errorf
			if sel.Sel.Name != "Errorf" {
				return true
			}

			// Check if the package is fmt
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "fmt" {
				return true
			}

			// Check if there are arguments
			if len(call.Args) == 0 {
				return true
			}

			// Check if the first argument is a string literal
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			// Get the string value
			formatStr := lit.Value

			// Check if the format string contains any placeholders (%v, %s, %d, %+v, etc.)
			// If it doesn't contain %, it's a fixed string and should use errors.New()
			if !strings.Contains(formatStr, "%") {
				pass.Reportf(call.Pos(), "%s: fixed error strings should use errors.New() instead of fmt.Errorf(): %s", analyzerName, formatStr)
			}

			return true
		})
	}

	return nil, nil
}
