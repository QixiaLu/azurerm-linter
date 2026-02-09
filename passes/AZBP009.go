package passes

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZBP009Doc = `check that variables do not shadow imported package names

The AZBP009 analyzer reports when variables are declared with the same name as imported packages,
which shadows the package import and makes it unusable.

Example violations:

	import "context"
	
	func badFunction() {
		context := "invalid"  // Shadows the context package
		// Now you can't use context.Background()
	}

	import "github.com/hashicorp/go-azure-helpers/lang/pointer"
	
	const pointer := "invalid"  // Shadows the pointer package

Correct usage:

	import "context"
	
	func goodFunction() {
		ctx := context.Background()  // Use different variable name
	}

	import "github.com/hashicorp/go-azure-helpers/lang/pointer"
	
	const pointerValue := "valid"  // Use different variable name
`

const azbp009Name = "AZBP009"

var AZBP009Analyzer = &analysis.Analyzer{
	Name:     azbp009Name,
	Doc:      AZBP009Doc,
	Run:      runAZBP009,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP009(pass *analysis.Pass) (interface{}, error) {
	if helper.IsCachePath(pass.Pkg.Path()) {
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

	// Create a map of import names per file
	fileImports := make(map[string]map[string]bool)
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		importNames := make(map[string]bool)

		for _, imp := range file.Imports {
			var importName string

			if imp.Name != nil {
				// Explicitly aliased import (foo "example.com/pkg")
				// Exclude blank (_) and dot (.) imports
				if imp.Name.Name != "_" && imp.Name.Name != "." {
					importName = imp.Name.Name
				}
			} else if imp.Path != nil {
				// Regular import - infer name from package path
				path := imp.Path.Value
				if len(path) >= 2 {
					path = path[1 : len(path)-1]
					if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
						path = path[lastSlash+1:]
					}
					if path != "" {
						importName = path
					}
				}
			}

			if importName != "" {
				importNames[importName] = true
			}
		}
		fileImports[filename] = importNames
	}

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),    // var, const declarations
		(*ast.AssignStmt)(nil), // := assignments
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		pos := pass.Fset.Position(n.Pos())
		if !loader.ShouldReport(pos.Filename, pos.Line) {
			return
		}

		importNames := fileImports[pos.Filename]
		if importNames == nil {
			return
		}

		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Tok == token.VAR || node.Tok == token.CONST {
				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if importNames[name.Name] {
								if ignorer.ShouldIgnore(azbp009Name, name) {
									continue
								}
								pass.Reportf(name.Pos(), "%s: variable '%s' shadows imported package name\n",
									azbp009Name, helper.FixedCode(name.Name))
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if importNames[ident.Name] {
							if ignorer.ShouldIgnore(azbp009Name, ident) {
								continue
							}
							pass.Reportf(ident.Pos(), "%s: variable '%s' shadows imported package name\n",
								azbp009Name, helper.FixedCode(ident.Name))
						}
					}
				}
			}
		}
	})

	return nil, nil
}
