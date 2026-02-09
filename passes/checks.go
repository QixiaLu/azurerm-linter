package passes

import (
	"golang.org/x/tools/go/analysis"
)

// AllChecks contains all Analyzers that report issues
var AllChecks = []*analysis.Analyzer{
	// AZBP001Analyzer,
	// AZBP002Analyzer,
	// AZBP003Analyzer,
	// AZBP004Analyzer,
	// AZBP005Analyzer,
	// AZBP006Analyzer,
	// AZBP007Analyzer,
	// AZBP008Analyzer,
	// AZBP009Analyzer,
	// AZBP010Analyzer,
	// AZBP011Analyzer,

	// AZSD001Analyzer,
	// AZSD002Analyzer,
	// AZSD003Analyzer,
	// AZSD004Analyzer,

	// AZRN001Analyzer,

	// AZRE001Analyzer,

	// AZNR001Analyzer,
	// AZNR002Analyzer,
	// AZNR003Analyzer,
	// AZNR004Analyzer,
	// AZNR005Analyzer,
	AZNR006Analyzer,
}
