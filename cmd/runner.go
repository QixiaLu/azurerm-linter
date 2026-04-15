package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// ExitCode represents program exit codes
type ExitCode int

const (
	ExitSuccess     ExitCode = 0 // No issues found
	ExitIssuesFound ExitCode = 1 // Lint issues found
	ExitError       ExitCode = 2 // Tool error
)

type Runner struct {
	Config *Config
}

// NewRunner creates a new Runner with the given config
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		Config: cfg,
	}
}

// Run executes the linter and returns an exit code
func (r *Runner) Run(ctx context.Context) ExitCode {
	defer loader.CleanupWorktree()

	isJSON := r.Config.OutputFormat == "json"
	scopeMode := r.detectFilterMode()

	loaderOpts := loader.LoaderOptions{
		NoFilter:   r.Config.NoFilter,
		PRNumber:   r.Config.PRNumber,
		RemoteName: r.Config.RemoteName,
		BaseBranch: r.Config.BaseBranch,
		DiffFile:   r.Config.DiffFile,
	}

	_, err := loader.LoadChanges(loaderOpts)
	if err != nil {
		log.Printf("Warning: failed to load changed lines filter: %v", err)
	}

	// Determine package patterns to analyze
	patterns := r.Config.Patterns
	if loader.IsEnabled() {
		files, lines := loader.GetStats()
		log.Printf("Changed lines filter: tracking %d files with %d changed lines", files, lines)

		// If change tracking is enabled and no patterns specified, use changed packages
		if len(r.Config.Patterns) == 0 {
			changedPackages := loader.GetChangedPackages()
			if len(changedPackages) > 0 {
				patterns = changedPackages
				log.Printf("Auto-detected %d changed packages:", len(patterns))
				for _, pkg := range patterns {
					log.Printf("  %s", pkg)
				}
			}
		}
	}

	// Validate we have patterns to analyze
	if len(patterns) == 0 {
		if isJSON {
			r.emitJSON(StatusSuccess, scopeMode, patterns, nil)
		} else {
			log.Println("✓ no service package to analyze")
		}
		return ExitSuccess
	}

	log.Printf("Loading packages...")
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		if isJSON {
			r.emitJSON(StatusError, scopeMode, patterns, nil)
		} else {
			log.Printf("Error: failed to load packages: %v", err)
		}
		return ExitError
	}

	// Check for package loading errors
	var hasLoadErrors bool
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			log.Printf("Error: failed to load package: %v", err)
			hasLoadErrors = true
		}
	})
	if hasLoadErrors {
		if isJSON {
			r.emitJSON(StatusError, scopeMode, patterns, nil)
		}
		return ExitError
	}

	// Provide loaded packages to analyzers for cross-package schema resolution
	helper.SetGlobalPackages(pkgs)

	log.Printf("Running analysis...")
	graph, err := checker.Analyze(passes.AllChecks, pkgs, nil)
	if err != nil {
		if isJSON {
			r.emitJSON(StatusError, scopeMode, patterns, nil)
		} else {
			log.Printf("Error: analysis failed: %v", err)
		}
		return ExitError
	}

	// Collect and report diagnostics
	findings := r.collectFindings(graph)

	if isJSON {
		status := StatusSuccess
		if len(findings) > 0 {
			status = StatusIssues
		}
		r.emitJSON(status, scopeMode, patterns, findings)
	} else {
		for _, f := range findings {
			fmt.Printf("%s:%d: %s\n", f.Path, f.Line, f.Message)
		}
		if len(findings) > 0 {
			fmt.Printf("Found %d issue(s)\n", len(findings))
		} else {
			log.Printf("✓ Analysis completed successfully with no issues found")
		}
	}

	if len(findings) > 0 {
		return ExitIssuesFound
	}
	return ExitSuccess
}

// detectFilterMode returns the FilterMode based on the current config
func (r *Runner) detectFilterMode() FilterMode {
	switch {
	case r.Config.NoFilter:
		return ModeUnfiltered
	case r.Config.DiffFile != "":
		return ModeDiff
	case r.Config.PRNumber > 0:
		return ModePR
	default:
		return ModeLocal
	}
}

// collectFindings walks the analysis graph and returns deduplicated findings.
// Messages are always stripped of ANSI codes and normalized to relative paths.
func (r *Runner) collectFindings(graph *checker.Graph) []JSONFinding {
	var findings []JSONFinding
	// Deduplicate diagnostics by "file:line:column|message"
	// When Tests=true, the same source file may be analyzed in both main and test packages
	// (when user doesn't mark test pkg as *_test), causing identical diagnostics to appear twice
	seen := make(map[string]bool)

	for act := range graph.All() {
		if act.Err != nil {
			continue
		}

		for _, diag := range act.Diagnostics {
			pos := act.Package.Fset.Position(diag.Pos)
			key := fmt.Sprintf("%s:%d:%d|%s", pos.Filename, pos.Line, pos.Column, diag.Message)

			if seen[key] {
				continue
			}
			seen[key] = true

			findings = append(findings, JSONFinding{
				CheckID: act.Analyzer.Name,
				Path:    pos.Filename,
				Line:    pos.Line,
				Message: stripANSI(diag.Message),
			})
		}
	}
	return findings
}
