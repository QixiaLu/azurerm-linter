package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/qixialu/azurerm-linter/passes"
)

// Version is set at build time via -ldflags, or auto-detected from build info
var Version = ""

func init() {
	if Version != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		Version = "dev"
		return
	}

	version := info.Main.Version
	if version == "" || version == "(devel)" {
		version = "dev"
	}
	if idx := strings.Index(version, "-"); idx > 0 && strings.HasPrefix(version, "v") {
		version = version[:idx]
	}

	var vcsRevision, vcsTime string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 8 {
				vcsRevision = s.Value[:8]
			} else {
				vcsRevision = s.Value
			}
		case "vcs.time":
			vcsTime = s.Value
		}
	}

	Version = "version " + version + " built with " + info.GoVersion
	if vcsRevision != "" {
		Version += " from " + vcsRevision
	}
	if vcsTime != "" {
		Version += " on " + vcsTime
	}
}

// Config holds all configuration options for the linter
type Config struct {
	// Command options
	Patterns    []string
	ShowHelp    bool
	ShowVersion bool
	ListChecks  bool

	// Output options
	OutputFormat string

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
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")
	fs.BoolVar(&cfg.ListChecks, "list", false, "list all available checks")

	// Output flags
	fs.StringVar(&cfg.OutputFormat, "output", "text", "output format: text or json")

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

	args := fs.Args()
	if len(args) > 0 && args[0] == "version" {
		cfg.ShowVersion = true
		return cfg, nil
	}

	cfg.Patterns = args

	return cfg, nil
}

// ShortVersion returns a compact version string (e.g. "v0.4.2" or "dev")
// suitable for JSON output and programmatic use.
func ShortVersion() string {
	v := Version
	if strings.HasPrefix(v, "version ") {
		return strings.Fields(v)[1]
	}
	return v
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
