package main

import (
	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	var analyzers []*analysis.Analyzer
	analyzers = append(analyzers, passes.AllChecks...)
	multichecker.Main(analyzers...)
}