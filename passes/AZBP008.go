package passes

import (
	"go/ast"
	"go/types"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
)

const AZBP008Doc = `check that ValidateFunc uses PossibleValuesFor instead of manual enum listing

When validating SDK enum types, use PossibleValuesFor* functions instead of
manually listing enum values with string() conversions.

Example violation:
    ValidateFunc: validation.StringInSlice([]string{
        string(webapps.ManagedPipelineModeClassic),
        string(webapps.ManagedPipelineModeIntegrated),
    }, false)

Valid usage:
    ValidateFunc: validation.StringInSlice(webapps.PossibleValuesForManagedPipelineMode(), false)
`

const azbp008Name = "AZBP008"

var AZBP008Analyzer = &analysis.Analyzer{
	Name:     azbp008Name,
	Doc:      AZBP008Doc,
	Run:      runAZBP008,
	Requires: []*analysis.Analyzer{localschema.LocalAnalyzer, commentignore.Analyzer},
}

func runAZBP008(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	schemaInfoList, ok := pass.ResultOf[localschema.LocalAnalyzer].(localschema.LocalSchemaInfoList)
	if !ok {
		return nil, nil
	}

	for _, cached := range schemaInfoList {
		schemaInfo := cached.Info

		if ignorer.ShouldIgnore(azbp008Name, schemaInfo.AstCompositeLit) {
			continue
		}

		validateFuncKV := schemaInfo.Fields[schema.SchemaFieldValidateFunc]
		if validateFuncKV == nil {
			continue
		}

		// Check if it's validation.StringInSlice([]string{...}, ...)
		call, ok := validateFuncKV.Value.(*ast.CallExpr)
		if !ok || len(call.Args) < 1 {
			continue
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "StringInSlice" {
			continue
		}

		compLit, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			continue
		}

		enumPkg, enumType := findChangedSDKEnum(pass, compLit.Elts)
		if enumPkg == "" {
			continue
		}

		pass.Reportf(call.Pos(), "%s: use %s instead of %s\n",
			azbp008Name,
			helper.FixedCode(enumPkg+".PossibleValuesFor"+enumType+"()"),
			helper.IssueLine("manually listing enum values"),
		)
	}

	return nil, nil
}

func findChangedSDKEnum(pass *analysis.Pass, elts []ast.Expr) (string, string) {
	enumPkg, enumNamed := extractEnumType(pass, elts)
	if enumNamed == nil || len(elts) != countEnumConstants(enumNamed) {
		return "", ""
	}

	for _, elt := range elts {
		pos := pass.Fset.Position(elt.Pos())
		if loader.ShouldReport(pos.Filename, pos.Line) {
			return enumPkg, enumNamed.Obj().Name()
		}
	}
	return "", ""
}

// extractEnumType returns the enum package and type if all elements are the same SDK enum type.
//
// Example input (elts from StringInSlice):
//
//	[]string{
//	    string(virtualmachines.VirtualMachinePriorityTypesLow),     // elt[0]
//	    string(virtualmachines.VirtualMachinePriorityTypesRegular), // elt[1]
//	    string(virtualmachines.VirtualMachinePriorityTypesSpot),    // elt[2]
//	}
//
// Returns: ("virtualmachines", *types.Named for VirtualMachinePriorityTypes)
func extractEnumType(pass *analysis.Pass, elts []ast.Expr) (string, *types.Named) {
	var enumPkg string
	var enumNamed *types.Named

	for _, elt := range elts {
		call, ok := elt.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return "", nil
		}

		sel, ok := call.Args[0].(*ast.SelectorExpr)
		if !ok {
			return "", nil
		}

		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return "", nil
		}

		obj := pass.TypesInfo.Uses[sel.Sel]
		if obj == nil {
			return "", nil
		}

		constObj, ok := obj.(*types.Const)
		if !ok {
			return "", nil
		}

		named, ok := constObj.Type().(*types.Named)
		if !ok || !helper.IsAzureSDKEnumType(pass, named) {
			return "", nil
		}

		if enumNamed == nil {
			enumPkg = pkgIdent.Name
			enumNamed = named
		} else if !types.Identical(named, enumNamed) {
			return "", nil
		}
	}

	return enumPkg, enumNamed
}

func countEnumConstants(named *types.Named) int {
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return 0
	}

	count := 0
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if c, ok := obj.(*types.Const); ok && types.Identical(c.Type(), named) {
			count++
		}
	}
	return count
}
