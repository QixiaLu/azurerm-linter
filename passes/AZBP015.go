package passes

import (
	"go/ast"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const AZBP015Doc = `check that check.That().Key().HasValue() is unnecessary when ImportStep is used

The AZBP015 analyzer reports when test code uses
check.That(data.ResourceName).Key("...").HasValue("...") in a test function that
also calls data.ImportStep(). The ImportStep already validates that all attribute
values match those in the config on read, making explicit HasValue assertions
redundant.

HasValue is only flagged in functions that contain an ImportStep call. Test
functions without ImportStep (e.g. some data source tests) are not affected.

Example violations (function has ImportStep):

  func TestAccExampleResource_basic(t *testing.T) {
      // ...
      check.That(data.ResourceName).Key("shape").HasValue("Exadata.X11M"),  // flagged
      // ...
      data.ImportStep(),
  }

Valid usage:

  check.That(data.ResourceName).ExistsInAzure(r),
  check.That(data.ResourceName).Key("id").Exists(),
  check.That(data.ResourceName).Key("name").IsNotEmpty(),
  // HasValue in functions WITHOUT ImportStep is fine:
  check.That(data.ResourceName).Key("example_property").HasValue("bar"),
`

const azbp015Name = "AZBP015"

var AZBP015Analyzer = &analysis.Analyzer{
	Name:     azbp015Name,
	Doc:      AZBP015Doc,
	Run:      runAZBP015,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

func runAZBP015(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	// Process each test file, function by function
	for _, file := range pass.Files {
		filePos := pass.Fset.Position(file.Pos())
		if !strings.HasSuffix(filePos.Filename, "_test.go") {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil {
				continue
			}

			// Only flag HasValue in functions that contain ImportStep
			if !containsImportStep(funcDecl.Body) {
				continue
			}

			// Find and report check.That().Key().HasValue() calls
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				pos := pass.Fset.Position(callExpr.Pos())
				if !loader.IsFileChanged(pos.Filename) {
					return true
				}
				if ignorer.ShouldIgnore(azbp015Name, callExpr) {
					return true
				}

				// Check for the pattern: <expr>.HasValue(<arg>)
				hasValueSel, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok || hasValueSel.Sel.Name != "HasValue" {
					return true
				}

				// The receiver of HasValue should be a call to Key: <expr>.Key(<arg>)
				keyCall, ok := hasValueSel.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				keySel, ok := keyCall.Fun.(*ast.SelectorExpr)
				if !ok || keySel.Sel.Name != "Key" {
					return true
				}

				// The receiver of Key should be a call to That: <pkg>.That(<arg>)
				thatCall, ok := keySel.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				thatSel, ok := thatCall.Fun.(*ast.SelectorExpr)
				if !ok || thatSel.Sel.Name != "That" {
					return true
				}

				reporting.Reportf(pass, reporting.ReportOptions{
					Rule:          azbp015Name,
					ReportPos:     callExpr.Pos(),
					EvidenceFile:  pos.Filename,
					EvidenceLines: []int{pos.Line},
					MatchMode:     reporting.MatchModeExactAdded,
				}, "%s: check.That().Key().HasValue() is unnecessary when ImportStep is present - the ImportStep validates that all config values match.\n",
					azbp015Name)

				return true
			})
		}
	}

	return nil, nil
}

// containsImportStep checks whether the function body contains a call to ImportStep.
func containsImportStep(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "ImportStep" {
			found = true
			return false
		}
		return true
	})
	return found
}
