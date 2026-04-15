package loader

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// LocalGitLoader loads changes from local git repository
type LocalGitLoader struct {
	remoteName string
	baseBranch string
}

// Load loads changes from local git repository and returns a ChangeSet
func (l *LocalGitLoader) Load() (*ChangeSet, error) {
	cs := NewChangeSet()

	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	targetCommit, err := resolveForLocal(repo, l.remoteName, l.baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target: %w", err)
	}

	if err := processDiffWithWorktree(cs, targetCommit); err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}
	if err := addUntrackedFiles(cs); err != nil {
		return nil, fmt.Errorf("failed to include untracked files: %w", err)
	}

	log.Printf("✓ Found %d changed files with %d changed lines",
		len(cs.changedFiles), cs.getTotalChangedLines())

	return cs, nil
}

// processDiffWithWorktree compares a commit with the current worktree using git diff
func processDiffWithWorktree(cs *ChangeSet, diffRef string) error {
	cmd := exec.Command("git", "diff", "--no-ext-diff", diffRef)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run git diff: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	diffOutput := string(output)
	if diffOutput == "" {
		return nil
	}

	return cs.parseDiffOutput(diffOutput)
}

func addUntrackedFiles(cs *ChangeSet) error {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list untracked files: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	fileNames := strings.Fields(string(output))
	return markUntrackedFiles(cs, fileNames)
}

func markUntrackedFiles(cs *ChangeSet, fileNames []string) error {
	for _, fileName := range fileNames {
		normalizedPath := normalizeFilePath(fileName)
		if !isServiceFile(normalizedPath) {
			continue
		}

		content, err := os.ReadFile(fileName)
		if err != nil {
			return fmt.Errorf("failed to read untracked file %s: %w", fileName, err)
		}

		cs.changedFiles[normalizedPath] = true
		cs.newFiles[normalizedPath] = true

		lineCount := strings.Count(string(content), "\n")
		if len(content) > 0 && content[len(content)-1] != '\n' {
			lineCount++
		}

		if lineCount == 0 {
			continue
		}

		if cs.changedLines[normalizedPath] == nil {
			cs.changedLines[normalizedPath] = make(map[int]bool, lineCount)
		}
		for line := 1; line <= lineCount; line++ {
			cs.changedLines[normalizedPath][line] = true
		}
	}

	return nil
}

// resolveForLocal resolves the diff reference for comparison.
func resolveForLocal(repo *git.Repository, remoteName, baseBranch string) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("not on a branch (detached HEAD)")
	}

	currentBranch := head.Name().Short()
	log.Printf("Current branch: %s", currentBranch)

	targetRemote, targetBranch, err := detectTargetBranch(repo, currentBranch, remoteName, baseBranch)
	if err != nil {
		return "", err
	}

	// Verify the remote reference exists
	targetRefName := fmt.Sprintf("%s/%s", targetRemote, targetBranch)
	_, err = repo.Reference(
		plumbing.NewRemoteReferenceName(targetRemote, targetBranch),
		true,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %w (try running 'git fetch %s %s')", targetRefName, err, targetRemote, targetBranch)
	}

	// Use shell 'git merge-base' command for robust merge-base detection
	mergeBaseHash, err := getMergeBase(targetRefName, "HEAD")
	return resolveDiffReference(targetRefName, mergeBaseHash, err)
}

func resolveDiffReference(targetRefName, mergeBaseHash string, mergeBaseErr error) (string, error) {
	if mergeBaseErr != nil {
		log.Printf("Warning: git merge-base failed for %s, falling back to direct diff against that ref: %v", targetRefName, mergeBaseErr)
		return targetRefName, nil
	}

	log.Printf("Merge-base with %s: %s", targetRefName, mergeBaseHash[:7])
	return mergeBaseHash, nil
}

// getMergeBase uses 'git merge-base' command to find the common ancestor
func getMergeBase(ref1, ref2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref1, ref2)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("git merge-base failed: %s", outputStr)
		}
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}

	hash := strings.TrimSpace(string(output))
	if hash == "" {
		return "", fmt.Errorf("git merge-base returned empty result")
	}

	return hash, nil
}

// detectTargetBranch detects the target remote and branch for comparison
func detectTargetBranch(repo *git.Repository, currentBranch, remoteName, baseBranch string) (string, string, error) {
	var detectedRemote, detectedBranch string

	// Check user-specified options
	if remoteName != "" {
		detectedRemote = remoteName
	}
	if baseBranch != "" {
		detectedBranch = baseBranch
	}

	if detectedRemote == "" || detectedBranch == "" {
		if configRemote, configBranch, ok := getUpstreamFromConfig(repo, currentBranch); ok {
			if detectedRemote == "" {
				detectedRemote = configRemote
			}
			if detectedBranch == "" {
				detectedBranch = configBranch
			}
			if detectedRemote == configRemote && detectedBranch == configBranch {
				log.Printf("Using upstream from branch config: %s/%s", detectedRemote, detectedBranch)
			}
		}
	}

	if detectedRemote == "" {
		remote, err := autoDetectRemote(repo)
		if err != nil {
			return "", "", err
		}
		detectedRemote = remote
	}

	if detectedBranch == "" {
		detectedBranch = "main"
	}

	return detectedRemote, detectedBranch, nil
}

// getUpstreamFromConfig gets upstream remote and branch from git config
func getUpstreamFromConfig(repo *git.Repository, currentBranch string) (remote, branch string, ok bool) {
	branchConfig, err := repo.Branch(currentBranch)
	if err != nil || branchConfig.Remote == "" {
		return "", "", false
	}

	remote = branchConfig.Remote
	branch = branchConfig.Merge.Short()

	if branch == "" {
		return "", "", false
	}

	return remote, branch, true
}

// autoDetectRemote auto-detects the remote
func autoDetectRemote(repo *git.Repository) (string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to list remotes: %w", err)
	}

	var foundUpstream bool
	for _, remote := range remotes {
		name := remote.Config().Name
		if name == "origin" {
			return "origin", nil
		}
		if name == "upstream" {
			foundUpstream = true
		}
	}

	if foundUpstream {
		return "upstream", nil
	}

	return "", fmt.Errorf("no suitable remote found (origin or upstream)")
}
