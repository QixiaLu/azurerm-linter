package changedlines

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func initializeFromGit() error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Using git method (local mode)\n")
	baseCommit, headCommit, err := resolveForLocal(repo)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Comparing: %s..%s\n",
		baseCommit.Hash.String()[:7],
		headCommit.Hash.String()[:7])

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

		patch, err := change.Patch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get patch for %s: %v\n", filePath, err)
			continue
		}

		if err := parsePatch(filePath, patch.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", filePath, err)
		}

		changedFiles[filePath] = true
	}

	fmt.Fprintf(os.Stderr, "Found %d changed files with %d lines\n",
		len(changedFiles), getTotalChangedLines())

	return nil
}

func initializeFromDiffFile(filePath string) error {
	fmt.Fprintf(os.Stderr, "Reading diff from file: %s\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open diff file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentFile string
	diffGitRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	fileCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Match diff --git a/file b/file
		if matches := diffGitRegex.FindStringSubmatch(line); matches != nil {
			currentFile = matches[2]
			fileCount++
			if !isServiceFile(currentFile) {
				currentFile = ""
			} else {
				// Normalize the file path before storing
				normalizedPath := normalizeFilePath(currentFile)
				changedFiles[normalizedPath] = true
				currentFile = normalizedPath
				fmt.Fprintf(os.Stderr, "Processing file: %s\n", currentFile)
			}
			continue
		}

		// Match +++ b/path/to/file (standard unified diff format)
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			fileCount++
			if !isServiceFile(currentFile) {
				currentFile = ""
			} else {
				// Normalize the file path before storing
				normalizedPath := normalizeFilePath(currentFile)
				changedFiles[normalizedPath] = true
				currentFile = normalizedPath
				fmt.Fprintf(os.Stderr, "Processing file: %s\n", currentFile)
			}
			continue
		}

		// Parse hunk headers @@ -x,y +a,b @@
		if currentFile != "" && strings.HasPrefix(line, "@@") {
			if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
				startLine, _ := strconv.Atoi(matches[1])
				count := 1
				if matches[2] != "" {
					if c, _ := strconv.Atoi(matches[2]); c > 0 {
						count = c
					}
				}

				if changedLines[currentFile] == nil {
					changedLines[currentFile] = make(map[int]bool)
				}

				for i := 0; i < count; i++ {
					changedLines[currentFile][startLine+i] = true
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Total files in diff: %d\n", fileCount)

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read diff file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files with %d changed lines\n",
		len(changedFiles), getTotalChangedLines())

	return nil
}

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

	return targetCommit, headCommit, nil
}

func detectTargetBranch(repo *git.Repository, currentBranch string) (string, string, error) {
	var detectedRemote, detectedBranch string

	if remoteName != nil && *remoteName != "" {
		detectedRemote = *remoteName
		fmt.Fprintf(os.Stderr, "Using user-specified remote: %s\n", detectedRemote)
	}
	if baseBranch != nil && *baseBranch != "" {
		detectedBranch = *baseBranch
		fmt.Fprintf(os.Stderr, "Using user-specified branch: %s\n", detectedBranch)
	}

	if detectedRemote != "" && detectedBranch != "" {
		return detectedRemote, detectedBranch, nil
	}

	if detectedRemote == "" {
		branchConfig, err := repo.Branch(currentBranch)
		if err == nil && branchConfig.Remote != "" {
			detectedRemote = branchConfig.Remote
			if detectedBranch == "" && branchConfig.Merge.Short() != "" {
				detectedBranch = branchConfig.Merge.Short()
			}
			fmt.Fprintf(os.Stderr, "Using upstream from branch config: %s/%s\n", detectedRemote, detectedBranch)
		}
	}

	if detectedRemote == "" {
		remotes, err := repo.Remotes()
		if err != nil {
			return "", "", fmt.Errorf("failed to list remotes: %w", err)
		}

		for _, remote := range remotes {
			name := remote.Config().Name
			if name == "upstream" {
				detectedRemote = "upstream"
				break
			}
			if name == "origin" {
				detectedRemote = "origin"
			}
		}

		if detectedRemote == "" {
			return "", "", fmt.Errorf("no suitable remote found (origin or upstream)")
		}
	}

	if detectedBranch == "" {
		detectedBranch = "main"
	}

	return detectedRemote, detectedBranch, nil
}

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

func isServiceFile(path string) bool {
	return strings.Contains(path, servicePathPrefix)
}

func normalizeFilePath(filename string) string {
	normalizedFilename := filepath.ToSlash(filename)
	idx := strings.Index(normalizedFilename, servicePathPrefix)
	if idx < 0 {
		return normalizedFilename
	}
	return normalizedFilename[idx:]
}

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

func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized && len(changedLines) > 0
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	changedLines = nil
	changedFiles = nil
	initialized = false
}

func GetStats() (filesCount int, totalLines int) {
	mu.RLock()
	defer mu.RUnlock()

	filesCount = len(changedFiles)
	totalLines = getTotalChangedLines()
	return
}

func getTotalChangedLines() int {
	total := 0
	for _, lines := range changedLines {
		total += len(lines)
	}
	return total
}

func isGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func canUseGitHubAPI() bool {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	return os.Getenv("GITHUB_REPOSITORY") != "" &&
		(eventName == "pull_request" || eventName == "pull_request_target")
}

func initializeFromGitHubAPI() error {
	token := os.Getenv("GITHUB_TOKEN")
	owner, name, err := getRepoInfo()
	if err != nil {
		return err
	}

	prNum, err := getPRNumber()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Fetching PR #%d changes from GitHub API (%s/%s)...\n", prNum, owner, name)

	files, err := fetchPRFiles(token, owner, name, prNum)
	if err != nil {
		return fmt.Errorf("failed to fetch PR files: %w", err)
	}

	for _, file := range files {
		if !isServiceFile(file.Filename) {
			continue
		}

		if file.Patch != "" {
			if err := parsePatch(file.Filename, file.Patch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", file.Filename, err)
			}
		}

		changedFiles[file.Filename] = true
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files from GitHub API\n", len(changedFiles))
	return nil
}

type PRFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch"`
}

func fetchPRFiles(token, owner, name string, prNum int) ([]PRFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/files", owner, name, prNum)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var files []PRFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}

func getRepoInfo() (owner, name string, err error) {
	owner = "hashicorp"
	name = "terraform-provider-azurerm"

	if repoName != nil && *repoName != "" {
		name = *repoName
	}

	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner, name = parts[0], parts[1]
		}
	}

	return owner, name, nil
}

func getPRNumber() (int, error) {
	if prNumber != nil && *prNumber > 0 {
		return *prNumber, nil
	}

	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName == "pull_request" || eventName == "pull_request_target" {
		if eventPath := os.Getenv("GITHUB_EVENT_PATH"); eventPath != "" {
			data, err := os.ReadFile(eventPath)
			if err == nil {
				var event struct {
					Number      int `json:"number"`
					PullRequest struct {
						Number int `json:"number"`
					} `json:"pull_request"`
				}
				if err := json.Unmarshal(data, &event); err == nil {
					if event.Number > 0 {
						return event.Number, nil
					}
					if event.PullRequest.Number > 0 {
						return event.PullRequest.Number, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("could not determine PR number")
}
