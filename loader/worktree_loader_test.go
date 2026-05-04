package loader

import (
	"strings"
	"testing"
)

func TestFetchPRRefArgsAvoidsShallowFetch(t *testing.T) {
	loader := &WorktreeLoader{
		prNumber:   123,
		remoteName: "origin",
	}

	args := loader.fetchPRRefArgs()

	for _, arg := range args {
		if strings.HasPrefix(arg, "--depth=") {
			t.Fatalf("Do not do shallow fetch, it will also truncate the main worktree history. Expected args to not contain any '--depth=<n>' flag, got %v", args)
		}
	}
}
