package AZNR002

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

/**
1. Find all updatable properties from schema
2. Find Update function
3.1 model.* = *
3.2 use 'hasChange?`
4. start with inline, all in the same file
*/

var Analyzer = &analysis.Analyzer{
	Name:     "AZNR002",
	Doc:      "All updatable properties need to be in Update func or marked as forcenew",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(passes *analysis.Pass) (interface{}, error) {
	return nil, nil
}