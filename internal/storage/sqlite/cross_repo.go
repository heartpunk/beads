// Package sqlite implements cross-repository dependency resolution.
package sqlite

import (
	"context"
	"fmt"

	"github.com/steveyegge/beads/internal/types"
)

// ResolveQualifiedID resolves a qualified ID to a repository name and local ID
// Returns (repoName, localID, isLocal, error)
// For local IDs: ("", "bd-5", true, nil)
// For cross-repo: ("api", "bd-5", false, nil)
func (s *SQLiteStorage) ResolveQualifiedID(ctx context.Context, qualifiedID string) (string, string, bool, error) {
	parsed, err := types.ParseQualifiedID(qualifiedID)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to parse qualified ID: %w", err)
	}

	// Local reference
	if parsed.IsLocal {
		return "", parsed.IssueID, true, nil
	}

	// Cross-repo reference - need to resolve repo name
	var repoName string

	if parsed.RepoName != "" {
		// Direct repo name reference (e.g., "api:bd-5")
		repoName = parsed.RepoName
	} else if parsed.RemoteType != "" {
		// Remote reference (e.g., "gh:user/repo:bd-5")
		// Try to find matching repository
		foundRepo, found, err := s.FindRepositoryByRemote(ctx, parsed.RemoteType, parsed.RemoteValue)
		if err != nil {
			return "", "", false, fmt.Errorf("failed to find repository: %w", err)
		}
		if !found {
			// Repository not registered - graceful degradation
			// Store the qualified ID as-is
			return "", qualifiedID, false, nil
		}
		repoName = foundRepo
	}

	return repoName, parsed.IssueID, false, nil
}

// NormalizeQualifiedID returns the canonical form of a qualified ID
// This is used for storage - we always store in canonical form
func (s *SQLiteStorage) NormalizeQualifiedID(ctx context.Context, qualifiedID string) (string, error) {
	repoName, localID, isLocal, err := s.ResolveQualifiedID(ctx, qualifiedID)
	if err != nil {
		return "", err
	}

	if isLocal {
		return localID, nil
	}

	if repoName == "" {
		// Unresolved remote reference - store as-is
		return qualifiedID, nil
	}

	// Canonical form: repo:local-id
	return fmt.Sprintf("%s:%s", repoName, localID), nil
}

// GetCrossRepoDependencies returns all cross-repo dependencies (where repo_name is not NULL)
func (s *SQLiteStorage) GetCrossRepoDependencies(ctx context.Context) ([]*types.Dependency, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT issue_id, depends_on_id, repo_name, type, created_at, created_by
		FROM dependencies
		WHERE repo_name IS NOT NULL
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query cross-repo dependencies: %w", err)
	}
	defer rows.Close()

	var deps []*types.Dependency
	for rows.Next() {
		var dep types.Dependency
		var repoName *string
		err := rows.Scan(&dep.IssueID, &dep.DependsOnID, &repoName, &dep.Type, &dep.CreatedAt, &dep.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		deps = append(deps, &dep)
	}

	return deps, nil
}

// FormatDependencyForDisplay formats a dependency ID for display
// Returns the qualified ID if it's cross-repo, local ID otherwise
func (s *SQLiteStorage) FormatDependencyForDisplay(ctx context.Context, dependsOnID, repoName string) string {
	if repoName == "" {
		return dependsOnID
	}

	// If it already has the repo prefix, return as-is
	if parsed, err := types.ParseQualifiedID(dependsOnID); err == nil && !parsed.IsLocal {
		return dependsOnID
	}

	// Add repo prefix
	return fmt.Sprintf("%s:%s", repoName, dependsOnID)
}
