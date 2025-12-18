package loader

import (
	"bufio"
	"flag"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const servicePathPrefix = "internal/services/"

var (
	useGitRepo = flag.Bool("use-git-repo", true, "use git repository to calculate diff")
	remoteName = flag.String("remote", "", "remote name (default: auto-detect)")
	baseBranch = flag.String("base-branch", "", "base branch (default: main)")
	diffFile   = flag.String("diff-file", "", "path to a diff file to parse")

	useGitHubAPI = flag.Bool("use-github-api", false, "use GitHub API to get PR changes")
	prNumber     = flag.Int("pr-number", 0, "GitHub PR number")
	repoName     = flag.String("repo-name", "terraform-provider-azurerm", "GitHub repository name")

	hunkRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

	// globalChangeSet holds the current loaded ChangeSet
	// Set once by LoadChanges() before analyzers run, then only read by analyzers
	globalChangeSet *ChangeSet
)

// ChangeSet represents a set of changes loaded from a source
type ChangeSet struct {
	changedLines map[string]map[int]bool
	changedFiles map[string]bool
	newFiles     map[string]bool
}

// NewChangeSet creates a new empty ChangeSet
func NewChangeSet() *ChangeSet {
	return &ChangeSet{
		changedLines: make(map[string]map[int]bool),
		changedFiles: make(map[string]bool),
		newFiles:     make(map[string]bool),
	}
}

// ChangeLoader is an interface for loading git changes from different sources
type ChangeLoader interface {
	Load() (*ChangeSet, error)
}

// LoadChanges sets up the changed lines tracking system and returns a ChangeSet
func LoadChanges() (*ChangeSet, error) {
	var loader ChangeLoader

	// Check if user provided a diff file
	if diffFile != nil && *diffFile != "" {
		loader = &DiffFileLoader{filePath: *diffFile}
	} else if *useGitRepo {
		loader = selectGitLoader()
	}

	var cs *ChangeSet
	var err error

	if loader != nil {
		cs, err = loader.Load()
		if err != nil {
			return nil, err
		}
	} else {
		// Return empty ChangeSet if no loader is selected
		cs = NewChangeSet()
	}

	// Set global ChangeSet for package-level functions
	globalChangeSet = cs

	return cs, nil
}

// selectGitLoader selects the appropriate git-based loader
func selectGitLoader() ChangeLoader {
	if *useGitHubAPI && *prNumber > 0 {
		log.Printf("Using GitHub API for PR #%d", *prNumber)
		return &GitHubLoader{}
	}

	return &LocalGitLoader{}
}

// ShouldReport checks if a specific line in a file should be reported
func ShouldReport(filename string, line int) bool {
	if globalChangeSet == nil {
		return true
	}
	return globalChangeSet.ShouldReport(filename, line)
}

// IsFileChanged checks if a file has any changes
func IsFileChanged(filename string) bool {
	if globalChangeSet == nil {
		return true
	}
	return globalChangeSet.IsFileChanged(filename)
}

// IsNewFile checks if a file is newly added
func IsNewFile(filename string) bool {
	if globalChangeSet == nil {
		return true
	}
	return globalChangeSet.IsNewFile(filename)
}

// IsEnabled checks if change tracking is enabled and has data
func IsEnabled() bool {
	if globalChangeSet == nil {
		return false
	}
	return globalChangeSet.IsEnabled()
}

// GetStats returns statistics about tracked changes
func GetStats() (filesCount int, totalLines int) {
	if globalChangeSet == nil {
		return 0, 0
	}
	return globalChangeSet.GetStats()
}

// ChangeSet methods

// ShouldReport checks if a specific line in a file should be reported
func (cs *ChangeSet) ShouldReport(filename string, line int) bool {
	if len(cs.changedLines) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)

	if !isServiceFile(relPath) {
		return true
	}

	if lineMap, exists := cs.changedLines[relPath]; exists {
		return lineMap[line]
	}

	return false
}

// IsFileChanged checks if a file has any changes
func (cs *ChangeSet) IsFileChanged(filename string) bool {
	if len(cs.changedFiles) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)
	if !isServiceFile(relPath) {
		return true
	}

	return cs.changedFiles[relPath]
}

// IsNewFile checks if a file is newly added
func (cs *ChangeSet) IsNewFile(filename string) bool {
	if len(cs.newFiles) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)
	if !isServiceFile(relPath) {
		return true
	}

	return cs.newFiles[relPath]
}

// IsEnabled checks if change tracking is enabled and has data
func (cs *ChangeSet) IsEnabled() bool {
	return len(cs.changedLines) > 0
}

// GetStats returns statistics about tracked changes
func (cs *ChangeSet) GetStats() (filesCount int, totalLines int) {
	filesCount = len(cs.changedFiles)
	totalLines = cs.getTotalChangedLines()
	return
}

// getTotalChangedLines counts total changed lines across all files
func (cs *ChangeSet) getTotalChangedLines() int {
	total := 0
	for _, lines := range cs.changedLines {
		total += len(lines)
	}
	return total
}

// parsePatch parses a patch string and extracts changed line numbers into the ChangeSet
func (cs *ChangeSet) parsePatch(filePath string, patchContent string) error {
	scanner := bufio.NewScanner(strings.NewReader(patchContent))
	var currentLine int
	inHunk := false

	// Initialize the map once
	if cs.changedLines[filePath] == nil {
		cs.changedLines[filePath] = make(map[int]bool)
	}

	for scanner.Scan() {
		line := scanner.Text()

		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			startLine, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			currentLine = startLine
			inHunk = true
			continue
		}
		if !inHunk {
			continue
		}

		if len(line) == 0 {
			currentLine++
			continue
		}

		prefix := line[0]
		switch prefix {
		case '+':
			cs.changedLines[filePath][currentLine] = true
			currentLine++
		case ' ':
			currentLine++
		}
	}

	return scanner.Err()
}

// isServiceFile checks if a path is within the service directory
func isServiceFile(path string) bool {
	return strings.Contains(path, servicePathPrefix)
}

// normalizeFilePath normalizes a file path to a consistent format
func normalizeFilePath(filename string) string {
	normalizedFilename := filepath.ToSlash(filename)
	idx := strings.Index(normalizedFilename, servicePathPrefix)
	if idx < 0 {
		return normalizedFilename
	}
	return normalizedFilename[idx:]
}

// parseDiffOutput parses git diff output containing multiple files into the ChangeSet
func (cs *ChangeSet) parseDiffOutput(diffOutput string) error {
	diffGitRegex := regexp.MustCompile(`(?m)^diff --git a/(.+) b/(.+)$`)
	matches := diffGitRegex.FindAllStringSubmatchIndex(diffOutput, -1)
	isNewFileRegex := regexp.MustCompile(`(?m)^new file mode`)

	if len(matches) == 0 {
		return nil // No changes
	}

	for i, match := range matches {
		// Extract file path from the match (use b/ path which is the new path)
		fileName := diffOutput[match[4]:match[5]]

		if !isServiceFile(fileName) {
			continue
		}

		// Get the content of this file's diff (from this match to the next, or to the end)
		var patchContent string
		if i < len(matches)-1 {
			patchContent = diffOutput[match[0]:matches[i+1][0]]
		} else {
			patchContent = diffOutput[match[0]:]
		}

		normalizedPath := normalizeFilePath(fileName)

		isNewFile := isNewFileRegex.MatchString(patchContent)

		if err := cs.parsePatch(normalizedPath, patchContent); err != nil {
			log.Printf("Warning: failed to parse patch for %s: %v", normalizedPath, err)
			continue
		}

		cs.changedFiles[normalizedPath] = true
		if isNewFile {
			cs.newFiles[normalizedPath] = true
		}
	}

	return nil
}
