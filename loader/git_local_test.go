package loader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestMarkUntrackedFilesIncludesNewServiceFiles(t *testing.T) {
	root := t.TempDir()
	serviceFile := filepath.Join(root, "internal", "services", "cdn", "resource.go")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(serviceFile, []byte("package cdn\n\nfunc example() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	otherFile := filepath.Join(root, "README.md")
	if err := os.WriteFile(otherFile, []byte("ignored\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	cs := NewChangeSet()
	if err := markUntrackedFiles(cs, []string{
		filepath.Join("internal", "services", "cdn", "resource.go"),
		"README.md",
	}); err != nil {
		t.Fatalf("markUntrackedFiles() error = %v", err)
	}

	file := filepath.Join("repo", "internal", "services", "cdn", "resource.go")
	if !cs.IsFileChanged(file) {
		t.Fatalf("IsFileChanged() = false, want true")
	}
	if !cs.IsNewFile(file) {
		t.Fatalf("IsNewFile() = false, want true")
	}
	normalizedPath := normalizeFilePath(filepath.Join("internal", "services", "cdn", "resource.go"))
	if len(cs.changedLines[normalizedPath]) != 3 {
		t.Fatalf("changedLines count = %d, want 3", len(cs.changedLines[normalizedPath]))
	}
	if cs.IsFileChanged(filepath.Join("repo", "README.md")) {
		t.Fatalf("non-service file should not be marked changed")
	}
}

func TestDetectTargetBranchUsesConfiguredUpstreamWhenItDiffers(t *testing.T) {
	repo := newLocalGitLoaderTestRepo(t)
	setBranchConfig(t, repo, "feature/cdn", "origin", "main")

	remote, branch, err := detectTargetBranch(repo, "feature/cdn", "", "")
	if err != nil {
		t.Fatalf("detectTargetBranch() error = %v", err)
	}
	if remote != "origin" {
		t.Fatalf("remote = %q, want origin", remote)
	}
	if branch != "main" {
		t.Fatalf("branch = %q, want main", branch)
	}
}

func TestDetectTargetBranchRespectsExplicitBaseBranch(t *testing.T) {
	repo := newLocalGitLoaderTestRepo(t)
	setBranchConfig(t, repo, "feature/cdn", "origin", "feature/cdn")

	remote, branch, err := detectTargetBranch(repo, "feature/cdn", "origin", "release-2026")
	if err != nil {
		t.Fatalf("detectTargetBranch() error = %v", err)
	}
	if remote != "origin" {
		t.Fatalf("remote = %q, want origin", remote)
	}
	if branch != "release-2026" {
		t.Fatalf("branch = %q, want release-2026", branch)
	}
}

func TestResolveDiffReferenceUsesMergeBaseWhenAvailable(t *testing.T) {
	ref, err := resolveDiffReference("origin/main", "44aadb4abc123", nil)
	if err != nil {
		t.Fatalf("resolveDiffReference() error = %v", err)
	}
	if ref != "44aadb4abc123" {
		t.Fatalf("ref = %q, want merge-base hash", ref)
	}
}

func TestResolveDiffReferenceFallsBackToTargetRefWhenMergeBaseFails(t *testing.T) {
	ref, err := resolveDiffReference("origin/main", "", errors.New("exit status 1"))
	if err != nil {
		t.Fatalf("resolveDiffReference() error = %v", err)
	}
	if ref != "origin/main" {
		t.Fatalf("ref = %q, want origin/main", ref)
	}
}

func newLocalGitLoaderTestRepo(t *testing.T) *git.Repository {
	t.Helper()

	root := t.TempDir()
	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"https://example.com/repo.git"}}); err != nil {
		t.Fatalf("CreateRemote() error = %v", err)
	}

	return repo
}

func setBranchConfig(t *testing.T, repo *git.Repository, currentBranch, remoteName, mergeBranch string) {
	t.Helper()

	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("Config() error = %v", err)
	}
	cfg.Branches[currentBranch] = &config.Branch{
		Name:   currentBranch,
		Remote: remoteName,
		Merge:  plumbing.NewBranchReferenceName(mergeBranch),
	}
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}
