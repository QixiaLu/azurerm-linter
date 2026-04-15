package passes

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR005Doc = `check for alphabetically sorted registration map and slice entries

Registration methods in registration.go files should have their map entries and slice entries
sorted alphabetically for better maintainability and consistency.

Example violations:
func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{
		"azurerm_managed_disk":     nil,
		"azurerm_availability_set": nil, // should come first alphabetically
	}
}

func (r Registration) Resources() []sdk.Resource {
	return []sdk.Resource{
		WorkspaceResource{},
		ApiManagementResource{}, // should come first alphabetically
	}
}

Valid usage:
func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
	return map[string]*pluginsdk.Resource{
		"azurerm_availability_set": nil,
		"azurerm_managed_disk":     nil,
	}
}

func (r Registration) Resources() []sdk.Resource {
	return []sdk.Resource{
		ApiManagementResource{},
		WorkspaceResource{},
	}
}`

const aznr005Name = "AZNR005"

var AZNR005Analyzer = &analysis.Analyzer{
	Name:     aznr005Name,
	Doc:      AZNR005Doc,
	Run:      runAZNR005,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZNR005(pass *analysis.Pass) (interface{}, error) {
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

	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.Pos())
		if !strings.HasSuffix(filepath.Base(pos.Filename), "registration.go") {
			continue
		}
		if !loader.IsFileChanged(pos.Filename) {
			continue
		}

		inspector.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok {
				return
			}
			if !hasRegistrationReceiver(funcDecl) {
				return
			}

			if ignorer.ShouldIgnore(aznr005Name, funcDecl) {
				return
			}

			// Analyze the registration method for sorting violations
			analyzeRegistrationMethod(pass, funcDecl)
		})

		break
	}

	return nil, nil
}

// hasRegistrationReceiver checks if the function has a Registration receiver
func hasRegistrationReceiver(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}

	recv := funcDecl.Recv.List[0]
	var typeName string

	switch t := recv.Type.(type) {
	case *ast.Ident:
		typeName = t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			typeName = ident.Name
		}
	}

	return typeName == "Registration"
}

// analyzeRegistrationMethod examines registration methods for unsorted map or slice returns
func analyzeRegistrationMethod(pass *analysis.Pass, funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil {
		return
	}

	compositeLiteralsByObject := make(map[types.Object]*ast.CompositeLit)
	reported := make(map[token.Pos]bool)

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			recordCompositeLiteralDefinitionsForIdents(pass, compositeLiteralsByObject, node.Names, node.Values)
		case *ast.AssignStmt:
			recordCompositeLiteralDefinitions(pass, compositeLiteralsByObject, node.Lhs, node.Rhs)
		case *ast.ReturnStmt:
			for _, expr := range node.Results {
				reportCompositeLiteral(pass, compositeLiteralsByObject, reported, expr)
			}
		}
		return true
	})
}

func recordCompositeLiteralDefinitionsForIdents(pass *analysis.Pass, composites map[types.Object]*ast.CompositeLit, lhs []*ast.Ident, rhs []ast.Expr) {
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

		if obj := objectForIdent(pass, ident); obj != nil {
			composites[obj] = compositeLit
		}
	}
}

func recordCompositeLiteralDefinitions(pass *analysis.Pass, composites map[types.Object]*ast.CompositeLit, lhs []ast.Expr, rhs []ast.Expr) {
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

		if obj := objectForIdent(pass, ident); obj != nil {
			composites[obj] = compositeLit
		}
	}
}

func reportCompositeLiteral(pass *analysis.Pass, composites map[types.Object]*ast.CompositeLit, reported map[token.Pos]bool, expr ast.Expr) {
	if compositeLit, ok := expr.(*ast.CompositeLit); ok {
		reportSorting(pass, reported, compositeLit)
		return
	}

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return
	}

	obj := objectForIdent(pass, ident)
	if obj == nil {
		return
	}

	if compositeLit := composites[obj]; compositeLit != nil {
		reportSorting(pass, reported, compositeLit)
	}
}

func reportSorting(pass *analysis.Pass, reported map[token.Pos]bool, compositeLit *ast.CompositeLit) {
	if reported[compositeLit.Pos()] {
		return
	}
	if validateSorting(pass, compositeLit) {
		reported[compositeLit.Pos()] = true
	}
}

func objectForIdent(pass *analysis.Pass, ident *ast.Ident) types.Object {
	if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
		return obj
	}

	return pass.TypesInfo.Uses[ident]
}

// validateSorting examines composite literals for sorting violations
func validateSorting(pass *analysis.Pass, compositeLit *ast.CompositeLit) bool {
	if compositeLit.Type == nil {
		return false
	}

	typ := pass.TypesInfo.TypeOf(compositeLit)
	if typ == nil {
		return false
	}

	var registrations []string
	switch typ.Underlying().(type) {
	case *types.Map:
		registrations = extractRegistrationMapKeys(compositeLit)
	case *types.Slice:
		registrations = extractRegistrationResourceNames(compositeLit)
	default:
		return false
	}

	if sort.StringsAreSorted(registrations) {
		return false
	}

	evidenceLines := make([]int, 0, len(compositeLit.Elts))
	for _, elt := range compositeLit.Elts {
		pos := pass.Fset.Position(elt.Pos())
		evidenceLines = append(evidenceLines, pos.Line)
	}

	reporting.Report(pass, reporting.ReportOptions{
		Rule:          aznr005Name,
		ReportPos:     compositeLit.Pos(),
		Message:       aznr005Name + ": " + helper.FixedCode("registrations should be sorted alphabetically") + "\n",
		EvidenceFile:  pass.Fset.Position(compositeLit.Pos()).Filename,
		EvidenceLines: evidenceLines,
		MatchMode:     reporting.MatchModeSameHunk,
	})

	return true
}

// extractRegistrationMapKeys extracts resource name keys from registration map literals
func extractRegistrationMapKeys(compositeLit *ast.CompositeLit) []string {
	var keys []string
	for _, elt := range compositeLit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if basicLit, ok := kv.Key.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
				if key, err := strconv.Unquote(basicLit.Value); err == nil {
					keys = append(keys, key)
				}
			}
		}
	}
	return keys
}

// extractRegistrationResourceNames extracts resource struct names from registration slice literals
func extractRegistrationResourceNames(compositeLit *ast.CompositeLit) []string {
	var resourceNames []string
	for _, elt := range compositeLit.Elts {
		if compositeLit, ok := elt.(*ast.CompositeLit); ok {
			if ident, ok := compositeLit.Type.(*ast.Ident); ok {
				resourceNames = append(resourceNames, ident.Name)
			}
		}
	}
	return resourceNames
}
