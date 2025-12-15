package changedlines

import (
	"fmt"
	"os"
)

// initializeFromDiffFile initializes changed lines tracking from a diff file
func initializeFromDiffFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "Reading diff from file: %s\n", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read diff file: %w", err)
	}

	// Use the common parseDiffOutput function
	if err := parseDiffOutput(string(content)); err != nil {
		return err
	}

	if len(changedFiles) == 0 {
		return fmt.Errorf("no valid diff blocks found in file")
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files with %d changed lines\n",
		len(changedFiles), getTotalChangedLines())

	return nil
}
