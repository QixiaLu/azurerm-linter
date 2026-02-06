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

const AZBP006Doc = `check for redundant nil assignments to pointer fields in struct literals

The AZBP006 analyzer reports cases where pointer fields in struct literals are
explicitly initialized to nil. This is redundant because uninitialized pointer
fields automatically have their zero value (nil).

Note: This rule only checks pointer types, not slices/maps/interfaces.

Example violation:
  return &profiles.ProfileLogScrubbing{
      State:    &policyDisabled,
      Selector: nil,  // Redundant - Selector is *string
  }

Valid usage:
  return &profiles.ProfileLogScrubbing{
      State: &policyDisabled,
  }}`

const azbp006Name = "AZBP006"

var AZBP006Analyzer = &analysis.Analyzer{
	Name:     azbp006Name,
	Doc:      AZBP006Doc,
	Run:      runAZBP006,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP006(pass *analysis.Pass) (interface{}, error) {
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

	nodeFilter := []ast.Node{(*ast.CompositeLit)(nil)}
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		compositeLit, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		// Skip test files - nil in test tables is often semantically meaningful
		pos := pass.Fset.Position(compositeLit.Pos())
		if strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}

		structType := getStructType(pass.TypesInfo.TypeOf(compositeLit))
		if structType == nil {
			return
		}

		for _, elt := range compositeLit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			// Check if the value is nil
			ident, ok := kv.Value.(*ast.Ident)
			if !ok {
				continue
			}

			obj := pass.TypesInfo.Uses[ident]
			if _, isNil := obj.(*types.Nil); !isNil {
				continue
			}

			// Get field name
			keyIdent, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			fieldName := keyIdent.Name

			// Check if the field type is a pointer (not slice/map/interface)
			fieldObj, _, _ := types.LookupFieldOrMethod(structType, true, pass.Pkg, fieldName)
			field, isVar := fieldObj.(*types.Var)
			if !isVar || field == nil {
				continue
			}

			// Only flag pointer types, not slices/maps/interfaces
			if _, isPointer := field.Type().Underlying().(*types.Pointer); !isPointer {
				continue
			}

			pos := pass.Fset.Position(kv.Pos())
			if !loader.ShouldReport(pos.Filename, pos.Line) {
				continue
			}
			if ignorer.ShouldIgnore(azbp006Name, kv) {
				continue
			}
			pass.Reportf(kv.Pos(), "%s: redundant %s assignment to pointer field %q - %s\n",
				azbp006Name, helper.IssueLine("nil"), fieldName, helper.FixedCode("omit the field"))
		}
	})

	return nil, nil
}

// getStructType extracts the struct type from a type (handling pointers)
func getStructType(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	// Dereference pointer if needed
	if ptr, ok := t.Underlying().(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if _, ok := t.Underlying().(*types.Struct); ok {
		return t
	}
	return nil
}
