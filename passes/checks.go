package passes

import (
	"github.com/qixialu/azurerm-linter/passes/AZBP001"
	"github.com/qixialu/azurerm-linter/passes/AZBP002"
	"github.com/qixialu/azurerm-linter/passes/AZNR001"
	"github.com/qixialu/azurerm-linter/passes/AZRE001"
	"github.com/qixialu/azurerm-linter/passes/AZRN001"
	"github.com/qixialu/azurerm-linter/passes/AZSD001"
	"golang.org/x/tools/go/analysis"
)

// AllChecks contains all Analyzers that report issues
// This can be consumed via multichecker.Main(xpasses.AllChecks...) or by
// combining these Analyzers with additional custom Analyzers
var AllChecks = []*analysis.Analyzer{
	AZBP001.Analyzer,
	AZBP002.Analyzer,
	AZNR001.Analyzer,
	AZRE001.Analyzer,
	AZRN001.Analyzer,
	AZSD001.Analyzer,
}
