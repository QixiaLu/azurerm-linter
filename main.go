package main

import (
	"context"
	"fmt"
	"os"

	"github.com/qixialu/azurerm-linter/cmd"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse configuration from flags
	cfg, err := cmd.ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Usage: azurerm-linter [flags] <package patterns>\n")
		return 3
	}

	// Handle help flag
	if cfg.ShowHelp {
		cmd.PrintHelp()
		return 0
	}

	// Handle list checks flag
	if cfg.ListChecks {
		cmd.PrintChecks()
		return 0
	}

	// Create and run the linter
	runner := cmd.NewRunner(cfg)
	exitCode := runner.Run(context.Background())

	return int(exitCode)
}
