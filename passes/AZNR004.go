package passes

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR004Doc = `check that flatten functions returning slices do not return nil

The AZNR004 analyzer reports when flatten* functions that return a slice type
return nil instead of an empty slice.

Example violation:

	func flattenNetworkACLs(input *NetworkRuleSet) []NetworkACLs {
	    if input == nil {
	        return nil  // Should return []NetworkACLs{}
	    }
	    // ...
	}

Correct usage:

	func flattenNetworkACLs(input *NetworkRuleSet) []NetworkACLs {
	    if input == nil {
	        return []NetworkACLs{}  // Return empty slice
	    }
	    // ...
	}

	// Or using make:
	func flattenNetworkACLs(input *NetworkRuleSet) []NetworkACLs {
	    if input == nil {
	        return make([]NetworkACLs, 0)
	    }
	    // ...
	}
`

const aznr004Name = "AZNR004"

var AZNR004Analyzer = &analysis.Analyzer{
	Name:     aznr004Name,
	Doc:      AZNR004Doc,
	Run:      runAZNR004,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZNR004(pass *analysis.Pass) (interface{}, error) {
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

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			return
		}

		// Check if function name starts with "flatten" (case-insensitive)
		funcName := funcDecl.Name.Name
		if !strings.HasPrefix(strings.ToLower(funcName), "flatten") {
			return
		}

		// Check if function returns a slice type
		if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
			return
		}

		// Find ALL slice return types and their positions
		type sliceReturnInfo struct {
			index     int
			sliceType *ast.ArrayType
		}
		var sliceReturns []sliceReturnInfo
		for i, result := range funcDecl.Type.Results.List {
			if arr, ok := result.Type.(*ast.ArrayType); ok {
				sliceReturns = append(sliceReturns, sliceReturnInfo{index: i, sliceType: arr})
			}
		}

		if len(sliceReturns) == 0 {
			return
		}

		// Check function body for return statements returning nil
		if funcDecl.Body == nil {
			return
		}

		ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
			retStmt, ok := node.(*ast.ReturnStmt)
			if !ok {
				return true
			}

			// Check if return statement has results
			if len(retStmt.Results) == 0 {
				return true
			}

			// Check if any slice return position returns nil
			hasNilSlice := false
			for _, sr := range sliceReturns {
				if sr.index >= len(retStmt.Results) {
					continue
				}

				expr := retStmt.Results[sr.index]

				// Check if returning nil
				ident, ok := expr.(*ast.Ident)
				if !ok {
					continue
				}
				if _, isNil := pass.TypesInfo.Uses[ident].(*types.Nil); isNil {
					hasNilSlice = true
					break
				}
			}

			if !hasNilSlice {
				return true
			}

			// Check git filter
			pos := pass.Fset.Position(retStmt.Pos())
			if !loader.ShouldReport(pos.Filename, pos.Line) {
				return true
			}

			// Check comment ignore
			if ignorer.ShouldIgnore(aznr004Name, retStmt) {
				return true
			}

			pass.Reportf(retStmt.Pos(), "%s: flatten function '%s' should return an empty slice instead of %s\n",
				aznr004Name,
				funcName,
				helper.IssueLine("nil"))

			return true
		})
	})

	return nil, nil
}
