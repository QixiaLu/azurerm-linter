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
	initialized  bool

	hunkRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

// Initialize sets up the changed lines tracking system
func Initialize() error {
	mu.Lock()
	defer mu.Unlock()

	changedLines = make(map[string]map[int]bool)
	changedFiles = make(map[string]bool)

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

	for scanner.Scan() {
		line := scanner.Text()

		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			startLine, _ := strconv.Atoi(matches[1])
			count := 1
			if matches[2] != "" {
				if c, _ := strconv.Atoi(matches[2]); c > 0 {
					count = c
				}
			}

			if changedLines[filePath] == nil {
				changedLines[filePath] = make(map[int]bool)
			}

			for i := 0; i < count; i++ {
				changedLines[filePath][startLine+i] = true
			}
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

	if !isServiceFile(filename) {
		return true
	}

	relPath := normalizeFilePath(filename)
	return changedFiles[relPath]
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
