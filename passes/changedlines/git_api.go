package changedlines

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// PRFile represents a file in a GitHub PR
type PRFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch"`
}

// initializeFromGitHubAPI initializes changed lines tracking from GitHub API
func initializeFromGitHubAPI() error {
	token := os.Getenv("GITHUB_TOKEN")
	owner, name := getRepoInfo()

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

		// Normalize the file path for consistent tracking
		normalizedPath := normalizeFilePath(file.Filename)

		// Check if this is a new file
		isNewFile := file.Status == "added"

		if file.Patch != "" {
			if err := parsePatch(normalizedPath, file.Patch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse patch for %s: %v\n", file.Filename, err)
			}
		}

		changedFiles[normalizedPath] = true
		if isNewFile {
			newFiles[normalizedPath] = true
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Found %d changed files from GitHub API\n", len(changedFiles))
	return nil
}

// fetchPRFiles fetches the list of changed files from GitHub API
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("GitHub API returned status %d, failed to read body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var files []PRFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}

// getRepoInfo gets the repository owner and name
func getRepoInfo() (owner, name string) {
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

	return owner, name
}

// getPRNumber gets the PR number from flags or environment
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

// isGitHubActions checks if running in GitHub Actions
func isGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// canUseGitHubAPI checks if GitHub API can be used
func canUseGitHubAPI() bool {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	return os.Getenv("GITHUB_REPOSITORY") != "" &&
		(eventName == "pull_request" || eventName == "pull_request_target")
}
