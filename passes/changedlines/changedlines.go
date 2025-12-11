package changedlines

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	gitDiffFile = flag.String("git-diff", "", "path to git diff file, or '-' to read from stdin")

	mu               sync.RWMutex
	changedLines     map[string]map[int]bool // file -> line numbers
	changedFilesOnly map[string]bool         // files without specific lines
	initialized      bool
)

// Initialize sets up the changed files filter from git diff
func Initialize() error {
	mu.Lock()
	defer mu.Unlock()

	changedLines = make(map[string]map[int]bool)
	changedFilesOnly = make(map[string]bool)

	// If no git diff specified, disable filtering (check everything)
	if gitDiffFile == nil || *gitDiffFile == "" {
		initialized = true
		return nil
	}

	var reader io.Reader
	var closer io.Closer

	if *gitDiffFile == "-" {
		// Read from stdin
		reader = os.Stdin
	} else {
		// Read from file
		file, err := os.Open(*gitDiffFile)
		if err != nil {
			return fmt.Errorf("failed to open git diff file: %w", err)
		}
		reader = file
		closer = file
	}

	if closer != nil {
		defer closer.Close()
	}

	if err := parseGitDiff(reader); err != nil {
		return err
	}

	initialized = true
	return nil
}

// parseGitDiff parses git diff output and extracts changed lines
func parseGitDiff(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	var currentFile string

	// Regex to match diff headers and hunks
	// diff --git a/path b/path
	diffGitRegex := regexp.MustCompile(`^diff --git [ab]/(.+) [ab]/(.+)$`)
	// +++ b/internal/services/policy/client/client.go
	filePlusRegex := regexp.MustCompile(`^\+\+\+ b/(.+)$`)
	// @@ -12,0 +13 @@ or @@ -41,0 +44,6 @@
	hunkRegex := regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for +++ b/filename
		if matches := filePlusRegex.FindStringSubmatch(line); matches != nil {
			currentFile = matches[1]
			// Only track files in internal/services/
			if !strings.Contains(currentFile, "internal/services/") {
				currentFile = ""
			}
			continue
		}

		// Check for diff --git header as fallback
		if matches := diffGitRegex.FindStringSubmatch(line); matches != nil {
			// Use the second capture group (b/ path)
			currentFile = matches[2]
			// Only track files in internal/services/
			if !strings.Contains(currentFile, "internal/services/") {
				currentFile = ""
			}
			continue
		} // Check for hunk header (only if we're tracking this file)
		if currentFile != "" {
			if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
				startLine, _ := strconv.Atoi(matches[1])
				count := 1
				if matches[2] != "" {
					count, _ = strconv.Atoi(matches[2])
					if count == 0 {
						count = 1 // ",0" means 1 line was added
					}
				}

				// Ensure the file has a line map
				if changedLines[currentFile] == nil {
					changedLines[currentFile] = make(map[int]bool)
				}

				// Mark all lines in this range as changed
				for i := 0; i < count; i++ {
					changedLines[currentFile][startLine+i] = true
				}
			}
		}
	}

	return scanner.Err()
}

// ShouldReport returns true if the given file/line should be reported
func ShouldReport(filename string, line int) bool {
	mu.RLock()
	defer mu.RUnlock()

	// If not initialized or no files specified, report everything
	if !initialized || (len(changedLines) == 0 && len(changedFilesOnly) == 0) {
		return true
	}

	// Normalize the filename path
	normalizedFilename := filepath.ToSlash(filename)

	// Only filter files in internal/services/
	if !strings.Contains(normalizedFilename, "internal/services/") {
		// Files outside internal/services/ are always checked
		return true
	}

	// Extract the path relative to internal/services/
	idx := strings.Index(normalizedFilename, "internal/services/")
	if idx < 0 {
		return true
	}
	relPath := normalizedFilename[idx:]

	// Check if this file should be checked entirely
	if changedFilesOnly[relPath] {
		return true
	}

	// Check if this specific line was changed
	if lineMap, exists := changedLines[relPath]; exists {
		return lineMap[line]
	}

	// File not in changed list - don't report
	return false
}

// IsEnabled returns true if the changed lines filter is active
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized && (len(changedLines) > 0 || len(changedFilesOnly) > 0)
}

// Reset clears the changed lines filter
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	changedLines = nil
	changedFilesOnly = nil
	initialized = false
}

// GetStats returns statistics about the filter (for debugging)
func GetStats() (filesCount int, totalLines int) {
	mu.RLock()
	defer mu.RUnlock()

	filesCount = len(changedLines) + len(changedFilesOnly)
	for _, lines := range changedLines {
		totalLines += len(lines)
	}
	return
}
