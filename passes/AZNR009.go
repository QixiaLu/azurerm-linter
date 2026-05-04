package passes

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/reporting"

	"golang.org/x/tools/go/analysis"
)

const AZNR009Doc = `check that API response IDs are parsed before assigning to state

The AZNR009 analyzer reports when a Read function directly assigns an API
response ID field (e.g. resp.Properties.Subnet.Id) to a model state field
instead of parsing it first with a ParseXxxID() call.

Example violation:
  state.WebApplicationFirewallPolicyId = wafPolicy.Id

Valid usage:
  parsedId, err := wafpolicies.ParseID(wafPolicy.Id)
  state.WebApplicationFirewallPolicyId = parsedId.ID()`

const aznr009Name = "AZNR009"

var AZNR009Analyzer = &analysis.Analyzer{
	Name: aznr009Name,
	Doc:  AZNR009Doc,
	Run:  runAZNR009,
	Requires: []*analysis.Analyzer{
		schema.TypedResourceInfoAnalyzer,
		commentignore.Analyzer,
	},
}

func runAZNR009(pass *analysis.Pass) (interface{}, error) {
	if helper.ShouldSkipPackageForResourceAnalysis(pass.Pkg.Path()) {
		return nil, nil
	}

	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}
	allTypedResources, ok := pass.ResultOf[schema.TypedResourceInfoAnalyzer].([]*helper.TypedResourceInfo)
	if !ok {
		return nil, nil
	}

	for _, resource := range allTypedResources {
		if resource.ReadFunc == nil {
			continue
		}

		body := helper.GetFuncBody(pass, resource.ReadFunc)
		if body == nil {
			continue
		}

		modelTypeName := resource.ModelName
		ast.Inspect(body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				checkAssignStmt(pass, ignorer, resource, modelTypeName, node)
			case *ast.CompositeLit:
				checkCompositeLit(pass, ignorer, resource, modelTypeName, node)
			}
			return true
		})
	}

	return nil, nil
}

// checkAssignStmt checks: state.SomeFieldId = response.Props.Id
func checkAssignStmt(pass *analysis.Pass, ignorer *commentignore.Ignorer, resource *helper.TypedResourceInfo, modelTypeName string, assign *ast.AssignStmt) {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return
	}

	lhsSel, ok := assign.Lhs[0].(*ast.SelectorExpr)
	if !ok {
		return
	}

	fieldName := lhsSel.Sel.Name
	if !isIdFieldName(fieldName) || !helper.IsModelType(lhsSel.X, modelTypeName, resource.TypesInfo) {
		return
	}

	if isDirectIdAccess(assign.Rhs[0], modelTypeName, resource.TypesInfo) {
		reportDirectIdAssignment(pass, ignorer, resource, fieldName, assign.Pos(), assign.Rhs[0])
	}
}

// checkCompositeLit checks: ModelStruct{ SomeFieldId: response.Props.Id }
func checkCompositeLit(pass *analysis.Pass, ignorer *commentignore.Ignorer, resource *helper.TypedResourceInfo, modelTypeName string, compLit *ast.CompositeLit) {
	if !isCompositeLitModelType(compLit, modelTypeName, resource.TypesInfo) {
		return
	}

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if !isIdFieldName(keyIdent.Name) {
			continue
		}
		if isDirectIdAccess(kv.Value, modelTypeName, resource.TypesInfo) {
			reportDirectIdAssignment(pass, ignorer, resource, keyIdent.Name, kv.Pos(), kv.Value)
		}
	}
}

// isIdFieldName returns true for fields like "SubnetId" but not "Id" itself.
func isIdFieldName(name string) bool {
	return name != "Id" && name != "ID" &&
		(strings.HasSuffix(name, "Id") || strings.HasSuffix(name, "ID"))
}

// isDirectIdAccess returns true for selector expressions ending in .Id/.ID
// on non-model types. Handles pointer dereferences (*resp.Props.Id).
// Returns false for function calls (parsedId.ID()) and model-to-model copies.
func isDirectIdAccess(expr ast.Expr, modelTypeName string, typesInfo *types.Info) bool {
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		expr = starExpr.X
	}

	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return (sel.Sel.Name == "Id" || sel.Sel.Name == "ID") &&
		!helper.IsModelType(sel.X, modelTypeName, typesInfo)
}

func isCompositeLitModelType(compLit *ast.CompositeLit, modelTypeName string, typesInfo *types.Info) bool {
	if typesInfo != nil {
		if typ := typesInfo.TypeOf(compLit); typ != nil {
			if named, ok := typ.(*types.Named); ok {
				return named.Obj().Name() == modelTypeName
			}
		}
	}
	if compLit.Type == nil {
		return false
	}
	switch t := compLit.Type.(type) {
	case *ast.Ident:
		return t.Name == modelTypeName
	case *ast.SelectorExpr:
		return t.Sel.Name == modelTypeName
	}
	return false
}

func reportDirectIdAssignment(pass *analysis.Pass, ignorer *commentignore.Ignorer, resource *helper.TypedResourceInfo, fieldName string, pos token.Pos, rhsExpr ast.Expr) {
	if ignorer.ShouldIgnore(aznr009Name, rhsExpr) {
		return
	}

	position := pass.Fset.Position(pos)
	if !position.IsValid() {
		return
	}

	if !loader.IsFileChanged(position.Filename) {
		return
	}

	tfSchemaName := resource.ModelFieldToTFSchema[fieldName]
	if tfSchemaName == "" {
		tfSchemaName = fieldName
	}

	reporting.Report(pass, reporting.ReportOptions{
		Rule:      aznr009Name,
		ReportPos: pos,
		Message: aznr009Name + ": API response ID field `" +
			helper.IssueLine(tfSchemaName) +
			"` should be parsed (e.g. using " +
			helper.FixedCode("ParseXxxID()") +
			") before assigning to state\n",
		EvidenceFile:  position.Filename,
		EvidenceLines: []int{position.Line},
		MatchMode:     reporting.MatchModeExactAdded,
	})
}
