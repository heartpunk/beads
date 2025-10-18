// Package git provides utilities for Git repository detection and analysis.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// IsGitRepository checks if a directory is a git repository
func IsGitRepository(path string) (bool, error) {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check .git directory: %w", err)
	}
	return info.IsDir(), nil
}

// GetRemotes returns all git remotes for a repository
// Returns a map of remote name -> remote URL
func GetRemotes(repoPath string) (map[string]string, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git remotes: %w", err)
	}

	remotes := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "origin	https://github.com/user/repo.git (fetch)"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		remoteName := parts[0]
		remoteURL := parts[1]

		// Only store each remote once (skip duplicate fetch/push entries)
		if _, exists := remotes[remoteName]; !exists {
			remotes[remoteName] = remoteURL
		}
	}

	return remotes, nil
}

// GetCurrentBranch returns the current git branch
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RemoteInfo represents information about a git remote
type RemoteInfo struct {
	Name string
	URL  string
	Type string // "github", "gitlab", "git-url"
}

// AnalyzeRemote analyzes a git remote URL and returns structured information
func AnalyzeRemote(name, url string) *RemoteInfo {
	info := &RemoteInfo{
		Name: name,
		URL:  url,
		Type: "git-url", // default
	}

	// Check for GitHub
	if isGitHubURL(url) {
		info.Type = "github"
	} else if isGitLabURL(url) {
		info.Type = "gitlab"
	}

	return info
}

var (
	githubHTTPSRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(\.git)?$`)
	githubSSHRegex   = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(\.git)?$`)
	gitlabHTTPSRegex = regexp.MustCompile(`^https?://gitlab\.com/([^/]+)/([^/]+?)(\.git)?$`)
	gitlabSSHRegex   = regexp.MustCompile(`^git@gitlab\.com:([^/]+)/([^/]+?)(\.git)?$`)
)

// isGitHubURL checks if a URL is a GitHub URL
func isGitHubURL(url string) bool {
	return githubHTTPSRegex.MatchString(url) || githubSSHRegex.MatchString(url)
}

// isGitLabURL checks if a URL is a GitLab URL
func isGitLabURL(url string) bool {
	return gitlabHTTPSRegex.MatchString(url) || gitlabSSHRegex.MatchString(url)
}

// ExtractGitHubRepo extracts "user/repo" from a GitHub URL
// Returns empty string if not a GitHub URL
func ExtractGitHubRepo(url string) string {
	if matches := githubHTTPSRegex.FindStringSubmatch(url); matches != nil {
		return matches[1] + "/" + matches[2]
	}
	if matches := githubSSHRegex.FindStringSubmatch(url); matches != nil {
		return matches[1] + "/" + matches[2]
	}
	return ""
}

// ExtractGitLabRepo extracts "org/proj" from a GitLab URL
// Returns empty string if not a GitLab URL
func ExtractGitLabRepo(url string) string {
	if matches := gitlabHTTPSRegex.FindStringSubmatch(url); matches != nil {
		return matches[1] + "/" + matches[2]
	}
	if matches := gitlabSSHRegex.FindStringSubmatch(url); matches != nil {
		return matches[1] + "/" + matches[2]
	}
	return ""
}

// HasBeadsDirectory checks if a path contains a .beads directory
func HasBeadsDirectory(path string) (bool, error) {
	beadsDir := filepath.Join(path, ".beads")
	info, err := os.Stat(beadsDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check .beads directory: %w", err)
	}
	return info.IsDir(), nil
}

// DetectRepositoryIdentity analyzes a directory and returns information about it
type RepositoryIdentity struct {
	Path        string              // Absolute path to repository
	IsGitRepo   bool                // Is this a git repository?
	HasBeads    bool                // Does it have .beads directory?
	Remotes     map[string]string   // Remote name -> URL
	RemoteInfos []*RemoteInfo       // Analyzed remote information
	SuggestedName string            // Suggested short name for the repo
}

// DetectIdentity analyzes a directory and detects repository identity
func DetectIdentity(path string) (*RepositoryIdentity, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	identity := &RepositoryIdentity{
		Path: absPath,
	}

	// Check if it's a git repository
	isGit, err := IsGitRepository(absPath)
	if err != nil {
		return nil, err
	}
	identity.IsGitRepo = isGit

	// Check for .beads directory
	hasBeads, err := HasBeadsDirectory(absPath)
	if err != nil {
		return nil, err
	}
	identity.HasBeads = hasBeads

	// If it's a git repo, get remotes
	if isGit {
		remotes, err := GetRemotes(absPath)
		if err != nil {
			return nil, err
		}
		identity.Remotes = remotes

		// Analyze each remote
		for name, url := range remotes {
			info := AnalyzeRemote(name, url)
			identity.RemoteInfos = append(identity.RemoteInfos, info)
		}
	}

	// Suggest a name based on directory name
	identity.SuggestedName = filepath.Base(absPath)

	return identity, nil
}
