package passes

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func collectCompositeLiteralDefinitions(pass *analysis.Pass) map[types.Object]*ast.CompositeLit {
	composites := make(map[types.Object]*ast.CompositeLit)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ValueSpec:
				recordAssignedCompositeLiteralsForIdents(pass, composites, node.Names, node.Values)
			case *ast.AssignStmt:
				recordAssignedCompositeLiteralsForExprs(pass, composites, node.Lhs, node.Rhs)
			}
			return true
		})
	}

	return composites
}

func recordAssignedCompositeLiteralsForIdents(pass *analysis.Pass, composites map[types.Object]*ast.CompositeLit, lhs []*ast.Ident, rhs []ast.Expr) {
	for index, ident := range lhs {
		if index >= len(rhs) {
			break
		}
		if ident == nil || ident.Name == "_" {
			continue
		}

		compositeLit, ok := rhs[index].(*ast.CompositeLit)
		if !ok {
			continue
		}

		if obj := lookupTypesObject(pass, ident); obj != nil {
			composites[obj] = compositeLit
		}
	}
}

func recordAssignedCompositeLiteralsForExprs(pass *analysis.Pass, composites map[types.Object]*ast.CompositeLit, lhs []ast.Expr, rhs []ast.Expr) {
	for index, leftExpr := range lhs {
		if index >= len(rhs) {
			break
		}

		ident, ok := leftExpr.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		compositeLit, ok := rhs[index].(*ast.CompositeLit)
		if !ok {
			continue
		}

		if obj := lookupTypesObject(pass, ident); obj != nil {
			composites[obj] = compositeLit
		}
	}
}

func resolveCompositeLiteralExpr(pass *analysis.Pass, expr ast.Expr, composites map[types.Object]*ast.CompositeLit) *ast.CompositeLit {
	if compositeLit, ok := expr.(*ast.CompositeLit); ok {
		return compositeLit
	}

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}

	obj := lookupTypesObject(pass, ident)
	if obj == nil {
		return nil
	}

	return composites[obj]
}

func lookupTypesObject(pass *analysis.Pass, ident *ast.Ident) types.Object {
	if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
		return obj
	}

	return pass.TypesInfo.Uses[ident]
}

func compositeLiteralEvidence(compLit *ast.CompositeLit, fset *token.FileSet) (string, []int) {
	file := fset.Position(compLit.Pos()).Filename
	if len(compLit.Elts) == 0 {
		return file, []int{fset.Position(compLit.Pos()).Line}
	}

	lines := make([]int, 0, len(compLit.Elts))
	for _, elt := range compLit.Elts {
		lines = append(lines, fset.Position(elt.Pos()).Line)
	}

	return file, lines
}
