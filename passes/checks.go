package passes

import (
	"github.com/qixialu/azurerm-linter/passes/AZC001"
	"github.com/qixialu/azurerm-linter/passes/AZC002"
	"github.com/qixialu/azurerm-linter/passes/AZC003"
	"golang.org/x/tools/go/analysis"
)

// AllChecks contains all Analyzers that report issues
// This can be consumed via multichecker.Main(xpasses.AllChecks...) or by
// combining these Analyzers with additional custom Analyzers
var AllChecks = []*analysis.Analyzer{
	AZC001.Analyzer,
	AZC002.Analyzer,
	AZC003.Analyzer,
}
