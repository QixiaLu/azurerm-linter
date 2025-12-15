package changedlines

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initializeFromGit initializes changed lines tracking from local git repository
func initializeFromGit() error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Using git method (local mode)\n")
	baseCommit, headCommit, err := resolveForLocal(repo)
	if err != nil {
		return fmt.Errorf("failed to resolve commits: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Comparing: %s..%s\n",
		baseCommit.Hash.String()[:7],
		headCommit.Hash.String()[:7])

	if err := processDiff(baseCommit, headCommit); err != nil {
		return fmt.Errorf("failed to process diff: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files with %d changed lines\n",
		len(changedFiles), getTotalChangedLines())

	return nil
}

// processDiff processes the diff between two commits and populates changed files/lines
func processDiff(baseCommit, headCommit *object.Commit) error {
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get base tree: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get head tree: %w", err)
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	for _, change := range changes {
		filePath := change.To.Name
		if filePath == "" {
			filePath = change.From.Name
		}

		if !isServiceFile(filePath) {
			continue
		}

		normalizedPath := normalizeFilePath(filePath)

		patch, err := change.Patch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get patch for %s: %v\n", normalizedPath, err)
			continue
		}

		if err := parsePatch(normalizedPath, patch.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", normalizedPath, err)
			continue
		}

		changedFiles[normalizedPath] = true
	}

	return nil
}

// resolveForLocal resolves the base and head commits for local git comparison
func resolveForLocal(repo *git.Repository) (*object.Commit, *object.Commit, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return nil, nil, fmt.Errorf("not on a branch (detached HEAD)")
	}

	currentBranch := head.Name().Short()
	fmt.Fprintf(os.Stderr, "Current branch: %s\n", currentBranch)

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get head commit: %w", err)
	}

	targetRemote, targetBranch, err := detectTargetBranch(repo, currentBranch)
	if err != nil {
		return nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "Target: %s/%s\n", targetRemote, targetBranch)

	targetRef, err := repo.Reference(
		plumbing.NewRemoteReferenceName(targetRemote, targetBranch),
		true,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get %s/%s: %w", targetRemote, targetBranch, err)
	}

	targetCommit, err := repo.CommitObject(targetRef.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target commit: %w", err)
	}

	mergeBases, err := headCommit.MergeBase(targetCommit)
	if err != nil || len(mergeBases) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to find merge-base, using direct diff: %v\n", err)
		return targetCommit, headCommit, nil
	}

	mergeBase := mergeBases[0]
	fmt.Fprintf(os.Stderr, "Merge-base: %s\n", mergeBase.Hash.String()[:7])
	return mergeBase, headCommit, nil
}

// detectTargetBranch detects the target remote and branch for comparison
func detectTargetBranch(repo *git.Repository, currentBranch string) (string, string, error) {
	var detectedRemote, detectedBranch string

	if userRemote, userBranch, ok := getUserSpecifiedTarget(); ok {
		detectedRemote = userRemote
		detectedBranch = userBranch
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
				fmt.Fprintf(os.Stderr, "Using upstream from branch config: %s/%s\n", detectedRemote, detectedBranch)
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

	// 4. Use default branch if still not set
	if detectedBranch == "" {
		detectedBranch = "main"
	}

	return detectedRemote, detectedBranch, nil
}

// getUserSpecifiedTarget returns user-specified remote and branch from flags
func getUserSpecifiedTarget() (remote, branch string, ok bool) {
	hasRemote := remoteName != nil && *remoteName != ""
	hasBranch := baseBranch != nil && *baseBranch != ""

	if !hasRemote && !hasBranch {
		return "", "", false
	}

	if hasRemote {
		remote = *remoteName
		fmt.Fprintf(os.Stderr, "Using user-specified remote: %s\n", remote)
	}
	if hasBranch {
		branch = *baseBranch
		fmt.Fprintf(os.Stderr, "Using user-specified branch: %s\n", branch)
	}

	return remote, branch, true
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
