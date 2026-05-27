package passes

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
)

const AZSD005Doc = `check that "None" enum values are not listed in StringInSlice validation

The AZSD005 analyzer reports when validation.StringInSlice includes a "None"
enum value from the Azure SDK. Terraform has its own null type (field omission),
so listing "None" is superfluous. The provider is moving away from this pattern
and existing "None" values are planned for removal in version 4.0.

This applies both to manually listed enum constants and to PossibleValuesFor*
functions whose enum type includes a "None" constant.

Example violation (manual listing):
    ValidateFunc: validation.StringInSlice([]string{
        string(labplan.ShutdownOnIdleModeNone),
        string(labplan.ShutdownOnIdleModeUserAbsence),
        string(labplan.ShutdownOnIdleModeLowUsage),
    }, false)

Example violation (PossibleValuesFor* with None constant):
    ValidateFunc: validation.StringInSlice(labplan.PossibleValuesForShutdownOnIdleMode(), false)

Valid usage (exclude None, handle in Create/Read):
    ValidateFunc: validation.StringInSlice([]string{
        string(labplan.ShutdownOnIdleModeUserAbsence),
        string(labplan.ShutdownOnIdleModeLowUsage),
    }, false)
`

const azsd005Name = "AZSD005"

var AZSD005Analyzer = &analysis.Analyzer{
	Name: azsd005Name,
	Doc:  AZSD005Doc,
	Run:  runAZSD005,
	Requires: []*analysis.Analyzer{
		localschema.LocalAnalyzer,
		commentignore.Analyzer,
	},
}

func runAZSD005(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	schemaInfoList, ok := pass.ResultOf[localschema.LocalAnalyzer].(localschema.LocalSchemaInfoList)
	if !ok {
		return nil, nil
	}

	compositeLiteralsByObject := collectCompositeLiteralDefinitions(pass)

	for _, cached := range schemaInfoList {
		schemaInfo := cached.Info

		if ignorer.ShouldIgnore(azsd005Name, schemaInfo.AstCompositeLit) {
			continue
		}

		validateFuncKV := schemaInfo.Fields[schema.SchemaFieldValidateFunc]
		if validateFuncKV == nil {
			continue
		}

		// Check if it's validation.StringInSlice(...)
		call, ok := validateFuncKV.Value.(*ast.CallExpr)
		if !ok || len(call.Args) < 1 {
			continue
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "StringInSlice" {
			continue
		}

		// manual listing via composite literal
		if compLit := resolveCompositeLiteralExpr(pass, call.Args[0], compositeLiteralsByObject); compLit != nil {
			evidenceFile, _ := compositeLiteralEvidence(compLit, pass.Fset)
			if !loader.IsFileChanged(evidenceFile) {
				continue
			}
			if noneLines := findNoneConstantsInCompositeLit(pass, compLit); len(noneLines) > 0 {
				reporting.Reportf(pass, reporting.ReportOptions{
					Rule:          azsd005Name,
					ReportPos:     call.Pos(),
					EvidenceFile:  evidenceFile,
					EvidenceLines: noneLines,
					MatchMode:     reporting.MatchModeExactAdded,
				}, "%s: %s value should not be listed in %s\n",
					azsd005Name,
					helper.IssueLine("None"),
					helper.FixedCode("StringInSlice"),
				)
			}
			continue
		}

		// PossibleValuesFor* function call
		argCall, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		argSel, ok := argCall.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		funcName := argSel.Sel.Name
		if !strings.HasPrefix(funcName, "PossibleValuesFor") {
			continue
		}

		// Resolve the package of the function
		funcObj := pass.TypesInfo.Uses[argSel.Sel]
		if funcObj == nil {
			continue
		}
		funcPkg := funcObj.Pkg()
		if funcPkg == nil {
			continue
		}

		// Derive enum type name from function name
		enumTypeName := strings.TrimPrefix(funcName, "PossibleValuesFor")

		// Check if any constant of this enum type has value "None"
		if hasNoneConstant(pass, funcPkg, enumTypeName) {
			pos := pass.Fset.Position(call.Pos())
			if !loader.IsFileChanged(pos.Filename) {
				continue
			}
			reporting.Reportf(pass, reporting.ReportOptions{
				Rule:          azsd005Name,
				ReportPos:     call.Pos(),
				EvidenceFile:  pos.Filename,
				EvidenceLines: []int{pos.Line},
				MatchMode:     reporting.MatchModeExactAdded,
			}, "%s: %s includes a %s value - list values explicitly excluding \"None\" and handle it in Create/Read instead\n",
				azsd005Name,
				helper.IssueLine(funcName+"()"),
				helper.IssueLine("None"),
			)
		}
	}

	return nil, nil
}

// findNoneConstantsInCompositeLit inspects elements of a []string{...} composite literal
// and returns the line numbers of elements whose resolved constant value is "None" (case-insensitive).
// Only SDK enum constants (via string(pkg.Const) pattern) are checked.
func findNoneConstantsInCompositeLit(pass *analysis.Pass, compLit *ast.CompositeLit) []int {
	var noneLines []int

	for _, elt := range compLit.Elts {
		// Look for string(pkg.Const) pattern
		callExpr, ok := elt.(*ast.CallExpr)
		if !ok || len(callExpr.Args) != 1 {
			continue
		}

		selExpr, ok := callExpr.Args[0].(*ast.SelectorExpr)
		if !ok {
			continue
		}

		obj := pass.TypesInfo.Uses[selExpr.Sel]
		if obj == nil {
			continue
		}

		constObj, ok := obj.(*types.Const)
		if !ok {
			continue
		}

		// Check if this is an Azure SDK enum type
		named, ok := constObj.Type().(*types.Named)
		if !ok || !helper.IsAzureSDKEnumType(pass, named) {
			continue
		}

		// Get the constant's string value
		val := constObj.Val()
		if val.Kind() != constant.String {
			continue
		}
		strVal := constant.StringVal(val)

		if strings.EqualFold(strVal, "None") {
			line := pass.Fset.Position(elt.Pos()).Line
			noneLines = append(noneLines, line)
		}
	}

	return noneLines
}

// hasNoneConstant checks if a package has any constant of the given enum type
// whose string value is "None" (case-insensitive).
func hasNoneConstant(pass *analysis.Pass, pkg *types.Package, enumTypeName string) bool {
	scope := pkg.Scope()
	if scope == nil {
		return false
	}

	// Look up the named type
	typeObj := scope.Lookup(enumTypeName)
	if typeObj == nil {
		return false
	}
	typeName, ok := typeObj.(*types.TypeName)
	if !ok {
		return false
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return false
	}

	// Verify this is an Azure SDK enum type
	if !helper.IsAzureSDKEnumType(pass, named) {
		return false
	}

	// Scan all constants of this type
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		constObj, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		if !types.Identical(constObj.Type(), named) {
			continue
		}
		val := constObj.Val()
		if val.Kind() != constant.String {
			continue
		}
		strVal := constant.StringVal(val)
		if strings.EqualFold(strVal, "None") {
			return true
		}
	}

	return false
}
