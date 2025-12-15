package main

import (
    "flag"
    "fmt"
	"os"

    "github.com/qixialu/azurerm-linter/passes"
    "github.com/qixialu/azurerm-linter/passes/changedlines"
    "golang.org/x/tools/go/analysis"
    "golang.org/x/tools/go/analysis/multichecker"
)

func main() {
    // Parse flags first to get changed lines configuration
    flag.Parse()

    // Initialize changed lines filter
    if err := changedlines.Initialize(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to initialize changed lines filter: %v\n", err)
    }

    // Show filter stats if enabled
    if changedlines.IsEnabled() {
        files, lines := changedlines.GetStats()
        fmt.Fprintf(os.Stderr, "Changed lines filter: tracking %d files with %d changed lines\n", files, lines)
    }

    var analyzers []*analysis.Analyzer
    analyzers = append(analyzers, passes.AllChecks...)
    multichecker.Main(analyzers...)
}