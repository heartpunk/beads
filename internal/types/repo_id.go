// Package types defines core data structures for the bd issue tracker.
package types

import (
	"fmt"
	"regexp"
	"strings"
)

// QualifiedID represents a parsed issue reference that may span repositories
type QualifiedID struct {
	// RepoName is the short name of the repository (empty for current repo)
	RepoName string

	// IssueID is the issue identifier within the repository
	IssueID string

	// RemoteType is the type of remote reference (github, gitlab, git-url, local-path, or empty for repo name)
	RemoteType string

	// RemoteValue is the value for the remote (e.g., "user/repo" for GitHub, URL for git-url)
	RemoteValue string

	// IsLocal indicates this refers to the current repository
	IsLocal bool
}

// String returns the canonical string representation of the qualified ID
func (q *QualifiedID) String() string {
	if q.IsLocal {
		return q.IssueID
	}
	if q.RepoName != "" {
		return fmt.Sprintf("%s:%s", q.RepoName, q.IssueID)
	}
	if q.RemoteType != "" {
		return fmt.Sprintf("%s:%s:%s", q.RemoteType, q.RemoteValue, q.IssueID)
	}
	return q.IssueID
}

// Patterns for parsing various qualified ID formats
var (
	// Shorthand formats: gh:user/repo:issue-id or gl:org/proj:issue-id
	shorthandPattern = regexp.MustCompile(`^(gh|gl):([a-zA-Z0-9_\-]+/[a-zA-Z0-9_\-]+):(.+)$`)

	// Repository name format: repo-name:issue-id
	repoNamePattern = regexp.MustCompile(`^([a-zA-Z0-9_\-]+):(.+)$`)

	// Git URL formats: https://... or git@...:issue-id
	gitHTTPSPattern = regexp.MustCompile(`^(https://[^:]+):(.+)$`)
	gitSSHPattern   = regexp.MustCompile(`^(git@[^:]+):(.+)$`)
)

// ParseQualifiedID parses an issue reference into its components
// Supports multiple formats:
//   - "issue-id" - local repository
//   - "repo:issue-id" - named repository
//   - "gh:user/repo:issue-id" - GitHub shorthand
//   - "gl:org/proj:issue-id" - GitLab shorthand
//   - "https://github.com/user/repo.git:issue-id" - full HTTPS URL
//   - "git@github.com:user/repo.git:issue-id" - full SSH URL
func ParseQualifiedID(ref string) (*QualifiedID, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty reference")
	}

	// Check for shorthand format: gh:user/repo:issue-id or gl:org/proj:issue-id
	if matches := shorthandPattern.FindStringSubmatch(ref); matches != nil {
		remoteType := matches[1] // gh or gl
		remoteValue := matches[2] // user/repo
		issueID := matches[3]

		// Expand shorthand to full remote type
		var fullType string
		switch remoteType {
		case "gh":
			fullType = "github"
		case "gl":
			fullType = "gitlab"
		default:
			return nil, fmt.Errorf("unknown shorthand type: %s", remoteType)
		}

		return &QualifiedID{
			IssueID:     issueID,
			RemoteType:  fullType,
			RemoteValue: remoteValue,
			IsLocal:     false,
		}, nil
	}

	// Check for HTTPS URL format: https://github.com/user/repo.git:issue-id
	if matches := gitHTTPSPattern.FindStringSubmatch(ref); matches != nil {
		gitURL := matches[1]
		issueID := matches[2]

		return &QualifiedID{
			IssueID:     issueID,
			RemoteType:  "git-url",
			RemoteValue: gitURL,
			IsLocal:     false,
		}, nil
	}

	// Check for SSH URL format: git@github.com:user/repo.git:issue-id
	// Note: We need special handling here because SSH URLs already contain ':'
	// Format: git@host:path:issue-id
	if strings.HasPrefix(ref, "git@") {
		parts := strings.Split(ref, ":")
		if len(parts) >= 3 {
			// Last part is issue ID, everything before is the git SSH URL
			issueID := parts[len(parts)-1]
			gitURL := strings.Join(parts[:len(parts)-1], ":")

			return &QualifiedID{
				IssueID:     issueID,
				RemoteType:  "git-url",
				RemoteValue: gitURL,
				IsLocal:     false,
			}, nil
		}
	}

	// Check for repository name format: repo-name:issue-id
	if matches := repoNamePattern.FindStringSubmatch(ref); matches != nil {
		repoName := matches[1]
		issueID := matches[2]

		// If repo name looks like a shorthand we missed, that's an error
		if repoName == "gh" || repoName == "gl" {
			return nil, fmt.Errorf("incomplete shorthand reference: %s (expected format: %s:user/repo:issue-id)", ref, repoName)
		}

		return &QualifiedID{
			RepoName: repoName,
			IssueID:  issueID,
			IsLocal:  false,
		}, nil
	}

	// No colon found - this is a local issue ID
	return &QualifiedID{
		IssueID: ref,
		IsLocal: true,
	}, nil
}

// Repository represents metadata about a code repository
type Repository struct {
	Name        string
	Description string
	CreatedAt   string
	LastSynced  *string
}

// RepositoryRemote represents a remote location for a repository
type RepositoryRemote struct {
	ID          int
	RepoName    string
	RemoteType  string // 'local-path', 'github', 'gitlab', 'git-url'
	RemoteValue string // path, user/repo, or full URL
	IsPrimary   bool
	CreatedAt   string
}

// RemoteType constants
const (
	RemoteTypeLocalPath = "local-path"
	RemoteTypeGitHub    = "github"
	RemoteTypeGitLab    = "gitlab"
	RemoteTypeGitURL    = "git-url"
)

// IsValidRemoteType checks if a remote type is valid
func IsValidRemoteType(remoteType string) bool {
	switch remoteType {
	case RemoteTypeLocalPath, RemoteTypeGitHub, RemoteTypeGitLab, RemoteTypeGitURL:
		return true
	default:
		return false
	}
}

// NormalizeGitHubRemote normalizes GitHub remote values
// Converts various formats to canonical "user/repo" format:
//   - "user/repo" -> "user/repo"
//   - "https://github.com/user/repo" -> "user/repo"
//   - "https://github.com/user/repo.git" -> "user/repo"
//   - "git@github.com:user/repo.git" -> "user/repo"
func NormalizeGitHubRemote(value string) string {
	// Already in shorthand format
	if !strings.Contains(value, "/") || (!strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "git@")) {
		return value
	}

	// Remove protocol and domain
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimPrefix(value, "git@github.com:")

	// Remove .git suffix
	value = strings.TrimSuffix(value, ".git")

	return value
}

// NormalizeGitLabRemote normalizes GitLab remote values
// Converts various formats to canonical "org/proj" format:
//   - "org/proj" -> "org/proj"
//   - "https://gitlab.com/org/proj" -> "org/proj"
//   - "https://gitlab.com/org/proj.git" -> "org/proj"
//   - "git@gitlab.com:org/proj.git" -> "org/proj"
func NormalizeGitLabRemote(value string) string {
	// Already in shorthand format
	if !strings.Contains(value, "/") || (!strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "git@")) {
		return value
	}

	// Remove protocol and domain
	value = strings.TrimPrefix(value, "https://gitlab.com/")
	value = strings.TrimPrefix(value, "git@gitlab.com:")

	// Remove .git suffix
	value = strings.TrimSuffix(value, ".git")

	return value
}
