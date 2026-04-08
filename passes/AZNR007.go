package passes

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const AZNR007Doc = `check that resource names in test configurations start with "acctest"

The AZNR007 analyzer reports when top-level name attributes in HCL test
configurations do not start with "acctest". Only the first-level name attribute
(2-space indentation) inside a resource block is checked. Data source blocks
(data "..." "...") and nested block names (e.g. load_balancer name,
ip_restriction name) are not checked.

Example violations:

  name = "acckv%[1]d"
  name = "sdsds"
  name = "myresource%d"

Valid usage:

  name = "acctestkv%[1]d"
  name = "acctestresource%d"

Excluded resource types: azurerm_private_dns_zone (name is a domain, not a test identifier).
`

const aznr007Name = "AZNR007"

var AZNR007Analyzer = &analysis.Analyzer{
	Name:     aznr007Name,
	Doc:      AZNR007Doc,
	Run:      runAZNR007,
	Requires: []*analysis.Analyzer{inspect.Analyzer, commentignore.Analyzer},
}

// aznr007NameValueRegex matches top-level HCL name attributes (2-space indent) with a quoted string value.
// Uses multiline mode so ^ matches the start of each line within the string.
// Only matches name at exactly 2 spaces of indentation (top-level inside a resource block).
var aznr007NameValueRegex = regexp.MustCompile(`(?m)^  name\s*=\s*"([^"]+)"`)

var aznr007BlockDeclRegex = regexp.MustCompile(`(?m)^(\w+)\s+"([^"]+)"`)

var aznr007ExcludedResourceTypes = map[string]bool{
	"azurerm_private_dns_zone": true,
	"azurerm_subnet":           true,
}

func runAZNR007(pass *analysis.Pass) (interface{}, error) {
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

		// Only check test files
		if !strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}

		if ignorer.ShouldIgnore(aznr007Name, lit) {
			return
		}

		// Extract string value from the literal
		var value string
		isRawString := strings.HasPrefix(lit.Value, "`")
		if isRawString {
			value = lit.Value[1 : len(lit.Value)-1]
		} else {
			var err error
			value, err = strconv.Unquote(lit.Value)
			if err != nil {
				return
			}
		}

		// Find name = "..." patterns in the string
		matches := aznr007NameValueRegex.FindAllStringSubmatchIndex(value, -1)
		blockDecls := aznr007BlockDeclRegex.FindAllStringSubmatchIndex(value, -1)
		for _, loc := range matches {
			resourceType := enclosingResourceType(loc[0], blockDecls, value)
			if resourceType == "" {
				continue
			}
			if aznr007ExcludedResourceTypes[resourceType] {
				continue
			}

			nameValue := value[loc[2]:loc[3]]

			matchLine := pos.Line
			if isRawString {
				matchLine += strings.Count(value[:loc[0]], "\n")
			}

			if !loader.ShouldReport(pos.Filename, matchLine) {
				continue
			}

			if !strings.HasPrefix(nameValue, "acctest") {
				reportPos := lit.Pos()
				if isRawString && matchLine > pos.Line {
					reportPos = pass.Fset.File(lit.Pos()).LineStart(matchLine)
				}
				pass.Reportf(reportPos, "%s: resource name %q should start with %s\n",
					aznr007Name, nameValue,
					helper.FixedCode(`"acctest"`))
			}
		}
	})

	return nil, nil
}

// enclosingResourceType returns the resource type (e.g. "azurerm_resource_group") if the
// position is inside a resource block, or "" if it is not inside a resource block.
func enclosingResourceType(pos int, blockDecls [][]int, value string) string {
	for i := len(blockDecls) - 1; i >= 0; i-- {
		if blockDecls[i][0] < pos {
			if value[blockDecls[i][2]:blockDecls[i][3]] == "resource" {
				return value[blockDecls[i][4]:blockDecls[i][5]]
			}
			return ""
		}
	}
	return ""
}
