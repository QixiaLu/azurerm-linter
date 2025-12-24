package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/qixialu/azurerm-linter/passes"
)

var (
	showHelp   = flag.Bool("help", false, "show help message")
	listChecks = flag.Bool("list", false, "list all available checks")
)

// Config holds all configuration options for the linter
type Config struct {
	Patterns   []string
	ShowHelp   bool
	ListChecks bool
}

// ParseFlags parses command line flags and returns a Config
func ParseFlags() (*Config, error) {
	flag.Parse()

	cfg := &Config{
		Patterns:   flag.Args(),
		ShowHelp:   *showHelp,
		ListChecks: *listChecks,
	}

	return cfg, nil
}

// PrintHelp prints the help message
func PrintHelp() {
	fmt.Println(`azurerm-linter - AzureRM Provider code linting tool

Flags:`)
	flag.PrintDefaults()
}

// PrintChecks prints all available checks
func PrintChecks() {
	fmt.Println("Available checks:\n")
	for _, analyzer := range passes.AllChecks {
		title := strings.Split(analyzer.Doc, "\n")[0]
		fmt.Printf("  %-10s  %s\n", analyzer.Name, title)
	}
}
