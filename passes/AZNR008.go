package passes

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR008Doc = `check for hardcoded resource IDs in test configurations

The AZNR008 analyzer reports when HCL test configurations contain hardcoded
Azure resource IDs (e.g. /subscriptions/<GUID>/resourceGroups/.../providers/...).
Resource IDs should not be hardcoded because they tie tests to a specific
subscription and may reference resources that do not exist. Construct IDs
dynamically using a resource reference (e.g. azurerm_resource.test.id),
a data block (e.g. data.azurerm_client_config.current.subscription_id),
or fmt.Sprintf placeholders instead.

Example violations:

  source_id = "/subscriptions/049e5678-fbb1-4861-93f3-7528bd0779fd/resourceGroups/rg/providers/..."
  workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test/providers/..."

Valid usage:

  source_id = azurerm_resource.test.id
  webhook_resource_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/resourcegroups/..."
  source_id = %[2]s
`

const aznr008Name = "AZNR008"

var AZNR008Analyzer = &analysis.Analyzer{
	Name:     aznr008Name,
	Doc:      AZNR008Doc,
	Run:      runAZNR008,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

var aznr008HardcodedIdRegex = regexp.MustCompile(`/subscriptions/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/`)

func runAZNR008(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.BasicLit)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return
		}

		pos := pass.Fset.Position(lit.Pos())

		if !strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}

		if ignorer.ShouldIgnore(aznr008Name, lit) {
			return
		}

		var value string
		isRawString := strings.HasPrefix(lit.Value, "`")
		if isRawString {
			value = lit.Value[1 : len(lit.Value)-1]
		} else {
			return
		}

		matches := aznr008HardcodedIdRegex.FindAllStringIndex(value, -1)
		if len(matches) == 0 {
			return
		}

		loc := matches[0]
		matchLine := pos.Line
		if isRawString {
			matchLine += strings.Count(value[:loc[0]], "\n")
		}

		if !loader.ShouldReport(pos.Filename, matchLine) {
			return
		}

		pass.Reportf(lit.Pos(), "%s: do not %s in test configs\n",
			aznr008Name, helper.IssueLine("hardcode resource IDs"))
	})

	return nil, nil
}
