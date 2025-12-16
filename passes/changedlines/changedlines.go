package changedlines

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

	mu           sync.RWMutex
	changedLines map[string]map[int]bool
	changedFiles map[string]bool
	newFiles     map[string]bool
	initialized  bool

	hunkRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

// Initialize sets up the changed lines tracking system
func Initialize() error {
	mu.Lock()
	defer mu.Unlock()

	changedLines = make(map[string]map[int]bool)
	changedFiles = make(map[string]bool)
	newFiles = make(map[string]bool)

	// Check if user provided a diff file
	if diffFile != nil && *diffFile != "" {
		if err := initializeFromDiffFile(*diffFile); err != nil {
			return fmt.Errorf("failed to initialize from diff file: %w", err)
		}
	} else if useGitRepo != nil && *useGitRepo {
		if err := initializeFromGitRepoSmart(); err != nil {
			return fmt.Errorf("failed to initialize from git repo: %w", err)
		}
	}

	initialized = true
	return nil
}

// initializeFromGitRepoSmart chooses the best git-based initialization method
func initializeFromGitRepoSmart() error {
	if isGitHubActions() && canUseGitHubAPI() {
		fmt.Fprintf(os.Stderr, "Detected GitHub Actions with PR context\n")
		return initializeFromGitHubAPI()
	}

	if useGitHubAPI != nil && *useGitHubAPI {
		if prNumber != nil && *prNumber > 0 {
			fmt.Fprintf(os.Stderr, "Using GitHub API for PR #%d\n", *prNumber)
			return initializeFromGitHubAPI()
		}
		return fmt.Errorf("GitHub API mode requires -pr-number")
	}

	return initializeFromGit()
}

// parsePatch parses a patch string and extracts changed line numbers
func parsePatch(filePath string, patchContent string) error {
	scanner := bufio.NewScanner(strings.NewReader(patchContent))
	var currentLine int
	inHunk := false

	// Initialize the map once
	if changedLines[filePath] == nil {
		changedLines[filePath] = make(map[int]bool)
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
			changedLines[filePath][currentLine] = true
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

// ShouldReport checks if a specific line in a file should be reported
func ShouldReport(filename string, line int) bool {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized || len(changedLines) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)

	if !isServiceFile(relPath) {
		return true
	}

	if lineMap, exists := changedLines[relPath]; exists {
		return lineMap[line]
	}

	return false
}

// IsFileChanged checks if a file has any changes
func IsFileChanged(filename string) bool {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized || len(changedFiles) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)
	if !isServiceFile(relPath) {
		return true
	}

	return changedFiles[relPath]
}

// IsNewFile checks if a file is newly added
func IsNewFile(filename string) bool {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized || len(newFiles) == 0 {
		return true
	}

	relPath := normalizeFilePath(filename)
	if !isServiceFile(relPath) {
		return true
	}

	return newFiles[relPath]
}

// IsEnabled checks if change tracking is enabled and has data
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized && len(changedLines) > 0
}

// Reset clears all tracking data
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	changedLines = nil
	changedFiles = nil
	initialized = false
}

// GetStats returns statistics about tracked changes
func GetStats() (filesCount int, totalLines int) {
	mu.RLock()
	defer mu.RUnlock()

	filesCount = len(changedFiles)
	totalLines = getTotalChangedLines()
	return
}

// getTotalChangedLines counts total changed lines across all files
func getTotalChangedLines() int {
	total := 0
	for _, lines := range changedLines {
		total += len(lines)
	}
	return total
}

// parseDiffOutput parses git diff output containing multiple files
func parseDiffOutput(diffOutput string) error {
	diffGitRegex := regexp.MustCompile(`(?m)^diff --git a/(.+) b/(.+)$`)
	matches := diffGitRegex.FindAllStringSubmatchIndex(diffOutput, -1)

	if len(matches) == 0 {
		return nil // No changes
	}

	isNewFileRegex := regexp.MustCompile(`(?m)^new file mode`)

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

		if err := parsePatch(normalizedPath, patchContent); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", normalizedPath, err)
			continue
		}

		changedFiles[normalizedPath] = true
		if isNewFile {
			newFiles[normalizedPath] = true
		}
	}

	return nil
}
