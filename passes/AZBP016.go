package passes

import (
	"go/ast"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZBP016Doc = `check for usage of WaitForStateContext instead of custom pollers

The AZBP016 analyzer reports when code calls
WaitForStateContext. Going forward, the provider prefers custom pollers that
implement the pollers.PollerType interface.

Reference: https://github.com/hashicorp/terraform-provider-azurerm/pull/30066

Example violation:
  stateConf := &pluginsdk.StateChangeConf{
      Pending: []string{"Creating"},
      Target:  []string{"Created"},
      Refresh: refreshFunc,
      Timeout: 10 * time.Minute,
  }
  result, err := stateConf.WaitForStateContext(ctx)

Valid usage:
  pollerType := custompollers.NewMyCustomPoller(...)
  poller := pollers.NewPoller(pollerType, 10*time.Second, pollers.DefaultNumberOfDroppedConnectionsToAllow)
  if err := poller.PollUntilDone(ctx); err != nil {
      return err
  }
`

const azbp016Name = "AZBP016"

var AZBP016Analyzer = &analysis.Analyzer{
	Name:     azbp016Name,
	Doc:      AZBP016Doc,
	Run:      runAZBP016,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP016(pass *analysis.Pass) (interface{}, error) {
	if helper.ShouldSkipPackageForResourceAnalysis(pass.Pkg.Path()) {
		return nil, nil
	}

	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		pos := pass.Fset.Position(n.Pos())
		if !loader.IsNewFile(pos.Filename) {
			return
		}

		node, err := n.(*ast.CallExpr)
		if err {
			return
		}
		sel, ok := node.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		if sel.Sel.Name != "WaitForStateContext" {
			return
		}

		if ignorer.ShouldIgnore(azbp016Name, node) {
			return
		}
		reporting.Reportf(pass, reporting.ReportOptions{
			Rule:          azbp016Name,
			ReportPos:     node.Pos(),
			EvidenceFile:  pos.Filename,
			EvidenceLines: []int{pos.Line},
			MatchMode:     reporting.MatchModeNewFile,
		}, "%s: prefer custom pollers (implementing %s) over %s\n",
			azbp016Name,
			helper.FixedCode("pollers.PollerType"),
			helper.IssueLine("WaitForStateContext"),
		)
	})

	return nil, nil
}
