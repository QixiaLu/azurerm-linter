package loader

import (
	"fmt"
	"log"
	"os"
)

// DiffFileLoader loads changes from a diff file
type DiffFileLoader struct {
	filePath string
}

// Load loads changes from a diff file
func (l *DiffFileLoader) Load() error {
	log.Printf("Reading diff from file: %s", l.filePath)

	content, err := os.ReadFile(l.filePath)
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

	log.Printf("✓ Found %d changed files with %d changed lines",
		len(changedFiles), getTotalChangedLines())

	return nil
}
