package main

import (
	"flag"
	"log"

	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	// Parse flags first to get changed lines configuration
	flag.Parse()

	// Load changed lines filter
	if err := loader.LoadChanges(); err != nil {
		log.Printf("Warning: failed to load changed lines filter: %v", err)
	}

	// Show filter stats if enabled
	if loader.IsEnabled() {
		files, lines := loader.GetStats()
		log.Printf("Changed lines filter: tracking %d files with %d changed lines", files, lines)
	}

	var analyzers []*analysis.Analyzer
	analyzers = append(analyzers, passes.AllChecks...)
	multichecker.Main(analyzers...)
}
