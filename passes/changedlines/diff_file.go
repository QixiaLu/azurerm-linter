package changedlines

import (
	"fmt"
	"os"
	"regexp"
)

// initializeFromDiffFile initializes changed lines tracking from a diff file
func initializeFromDiffFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "Reading diff from file: %s\n", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read diff file: %w", err)
	}

	// Split diff content by file
	diffGitRegex := regexp.MustCompile(`(?m)^diff --git a/(.+) b/(.+)$`)
	matches := diffGitRegex.FindAllStringSubmatchIndex(string(content), -1)

	if len(matches) == 0 {
		return fmt.Errorf("no valid diff blocks found in file")
	}

	for i, match := range matches {
		// Extract file path from the match
		fileName := string(content[match[4]:match[5]])

		if !isServiceFile(fileName) {
			continue
		}

		// Get the content of this file's diff (from this match to the next, or to the end)
		var patchContent string
		if i < len(matches)-1 {
			patchContent = string(content[match[0]:matches[i+1][0]])
		} else {
			patchContent = string(content[match[0]:])
		}

		// Normalize the file path
		normalizedPath := normalizeFilePath(fileName)

		// Parse the patch for this file
		if err := parsePatch(normalizedPath, patchContent); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", normalizedPath, err)
			continue
		}

		changedFiles[normalizedPath] = true
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files with %d changed lines\n",
		len(changedFiles), getTotalChangedLines())

	return nil
}
