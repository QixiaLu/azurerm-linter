package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/qixialu/azurerm-linter/passes"
)

// Config holds all configuration options for the linter
type Config struct {
	// Command options
	Patterns   []string
	ShowHelp   bool
	ListChecks bool

	// Loader options
	NoFilter   bool
	PRNumber   int
	RemoteName string
	BaseBranch string
	DiffFile   string

	// Internal: flagSet for help printing
	flagSet *flag.FlagSet
}

// ParseFlags parses command line flags and returns a Config
func ParseFlags() (*Config, error) {
	fs := flag.NewFlagSet("azurerm-linter", flag.ExitOnError)

	// Config struct to populate
	cfg := &Config{flagSet: fs}

	// Command flags
	fs.BoolVar(&cfg.ShowHelp, "help", false, "show help message")
	fs.BoolVar(&cfg.ListChecks, "list", false, "list all available checks")

	// Loader flags
	fs.BoolVar(&cfg.NoFilter, "no-filter", false, "disable change filtering, analyze all files")
	fs.IntVar(&cfg.PRNumber, "pr", 0, "analyze GitHub PR by number")
	fs.StringVar(&cfg.RemoteName, "remote", "", "git remote name (auto-detect: origin > upstream)")
	fs.StringVar(&cfg.BaseBranch, "base", "", "base branch (auto-detect from git config or 'main')")
	fs.StringVar(&cfg.DiffFile, "diff", "", "read diff from file instead of git")

	fs.Usage = func() {
		cfg.PrintHelp()
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	cfg.Patterns = fs.Args()

	return cfg, nil
}

// PrintHelp prints the help message
func (c *Config) PrintHelp() {
	fmt.Println(`azurerm-linter - AzureRM Provider code linting tool

Usage:
  azurerm-linter [flags] <package patterns>

Examples:
  azurerm-linter ./internal/services/compute/...
  azurerm-linter --pr=12345
  azurerm-linter --diff=changes.txt
  azurerm-linter --no-filter ./internal/services/...

Flags:`)
	c.flagSet.PrintDefaults()
}

// PrintChecks prints all available checks
func PrintChecks() {
	fmt.Println("Available checks:")
	for _, analyzer := range passes.AllChecks {
		title := strings.Split(analyzer.Doc, "\n")[0]
		fmt.Printf("  %-10s  %s\n", analyzer.Name, title)
	}
}
