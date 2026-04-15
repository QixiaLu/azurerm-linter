package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/qixialu/azurerm-linter/loader"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// FilterMode describes how the analysis scope was determined
type FilterMode string

// Status describes the outcome of the linter run
type Status string

const (
	ModeUnfiltered FilterMode = "unfiltered" // --no-filter: all files analyzed
	ModeDiff       FilterMode = "diff"       // --diff: changes from a diff file
	ModePR         FilterMode = "pr"         // --pr: changes from a GitHub PR
	ModeLocal      FilterMode = "local"      // default: local git diff

	StatusSuccess Status = "success"
	StatusIssues  Status = "issues_found"
	StatusError   Status = "error"
)

// JSONOutput is the top-level JSON output structure
type JSONOutput struct {
	Version  string        `json:"version"`
	Status   Status        `json:"status"`
	Scope    JSONScope     `json:"scope"`
	Summary  JSONSummary   `json:"summary"`
	Findings []JSONFinding `json:"findings"`
}

// JSONScope describes the analysis scope
type JSONScope struct {
	Mode     FilterMode `json:"mode"`
	Patterns []string   `json:"patterns"`
}

// JSONSummary holds aggregate counts
type JSONSummary struct {
	ChangedFiles int `json:"changed_files"`
	ChangedLines int `json:"changed_lines"`
	IssueCount   int `json:"issue_count"`
}

// JSONFinding represents a single diagnostic finding
type JSONFinding struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// emitJSON writes the JSON envelope to stdout
func (r *Runner) emitJSON(status Status, mode FilterMode, patterns []string, findings []JSONFinding) {
	if findings == nil {
		findings = []JSONFinding{}
	}
	if patterns == nil {
		patterns = []string{}
	}

	var changedFiles, changedLines int
	if loader.IsEnabled() {
		changedFiles, changedLines = loader.GetStats()
	}

	output := JSONOutput{
		Version: ShortVersion(),
		Status:  status,
		Scope: JSONScope{
			Mode:     mode,
			Patterns: patterns,
		},
		Summary: JSONSummary{
			ChangedFiles: changedFiles,
			ChangedLines: changedLines,
			IssueCount:   len(findings),
		},
		Findings: findings,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON output: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// stripANSI removes ANSI escape codes and trims whitespace from a string
func stripANSI(s string) string {
	return strings.TrimSpace(ansiRegex.ReplaceAllString(s, ""))
}
