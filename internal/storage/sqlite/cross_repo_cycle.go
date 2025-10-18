// Package sqlite implements cross-repository cycle detection.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/steveyegge/beads/internal/types"
)

// detectCrossRepoCycle checks for cycles across repositories by temporarily
// attaching the remote repository's database and querying the combined dependency graph.
//
// This is a proof of concept using SQLite's ATTACH DATABASE feature.
func (s *SQLiteStorage) detectCrossRepoCycle(ctx context.Context, tx *sql.Tx, dep *types.Dependency, repoName string) (bool, error) {
	// Get the remote repository to find its database path
	repo, err := s.GetRepository(ctx, repoName)
	if err != nil {
		return false, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		// Repository not registered - can't check for cycles
		return false, nil
	}

	// Get the primary local-path remote for the repository
	remotes, err := s.GetRepositoryRemotes(ctx, repoName)
	if err != nil {
		return false, fmt.Errorf("failed to get remotes: %w", err)
	}

	var dbPath string
	for _, remote := range remotes {
		if remote.RemoteType == types.RemoteTypeLocalPath && remote.IsPrimary {
			// Construct expected database path
			dbPath = filepath.Join(remote.RemoteValue, ".beads", repoName+".db")
			break
		}
	}

	if dbPath == "" {
		// No local path - can't attach database
		return false, nil
	}

	// Check if database exists
	// Note: In production, we might want to handle missing databases gracefully

	// ATTACH the remote database
	// Use a temporary alias to avoid conflicts
	attachAlias := fmt.Sprintf("remote_%s", repoName)
	_, err = tx.ExecContext(ctx, fmt.Sprintf("ATTACH DATABASE ? AS %s", attachAlias), dbPath)
	if err != nil {
		// If we can't attach, we can't verify - err on the side of allowing
		// (Could be missing database, permissions, etc.)
		return false, nil
	}
	defer tx.ExecContext(ctx, fmt.Sprintf("DETACH DATABASE %s", attachAlias))

	// Now run cycle detection across BOTH databases
	// The query unions dependencies from:
	// - main database (current repo)
	// - attached database (remote repo)
	//
	// For cross-repo dependencies, we need to:
	// 1. Map qualified IDs to their canonical form
	// 2. Handle bidirectional references (bd:bd-1 in api db, api:bd-5 in bd db)

	var cycleExists bool
	query := fmt.Sprintf(`
		WITH RECURSIVE
		-- Combine dependencies from both databases
		all_deps AS (
			-- Local dependencies (from main database)
			SELECT issue_id, depends_on_id FROM main.dependencies

			UNION ALL

			-- Remote dependencies (from attached database)
			-- Prefix remote issue IDs with repo name for proper matching
			SELECT
				? || ':' || issue_id as issue_id,  -- e.g., "api:bd-5"
				CASE
					-- If depends_on_id has a repo_name, it's cross-repo - keep as-is
					WHEN repo_name IS NOT NULL THEN depends_on_id
					-- Otherwise it's local to remote repo - add repo prefix
					ELSE ? || ':' || depends_on_id
				END as depends_on_id
			FROM %s.dependencies
		),
		paths AS (
			SELECT
				issue_id,
				depends_on_id,
				1 as depth
			FROM all_deps
			WHERE issue_id = ?

			UNION ALL

			SELECT
				d.issue_id,
				d.depends_on_id,
				p.depth + 1
			FROM all_deps d
			JOIN paths p ON d.issue_id = p.depends_on_id
			WHERE p.depth < ?
		)
		SELECT EXISTS(
			SELECT 1 FROM paths
			WHERE depends_on_id = ?
		)
	`, attachAlias)

	err = tx.QueryRowContext(ctx, query,
		repoName, repoName,  // For prefixing remote IDs
		dep.DependsOnID,     // Start from depends_on_id
		maxDependencyDepth,  // Max depth
		dep.IssueID,         // Check if we can reach issue_id
	).Scan(&cycleExists)

	if err != nil {
		return false, fmt.Errorf("failed to check for cross-repo cycles: %w", err)
	}

	return cycleExists, nil
}

// detectCrossRepoCycleMultiRepo checks for cycles when the dependency might span
// multiple repositories. This is a more general version that could handle complex
// multi-repo dependency chains.
//
// Example: bd-1 → api:bd-5 → cache:bd-3 → bd-1
//
// For the proof of concept, we'll keep it simple and just check the immediate
// remote repository.
func (s *SQLiteStorage) detectCrossRepoCycleMultiRepo(ctx context.Context, tx *sql.Tx, dep *types.Dependency, repoName string) (bool, error) {
	// This would involve:
	// 1. Identifying all repositories involved in the dependency chain
	// 2. Attaching all their databases
	// 3. Running a unified cycle detection query
	// 4. Detaching all databases
	//
	// For now, we'll just delegate to the simpler version
	return s.detectCrossRepoCycle(ctx, tx, dep, repoName)
}
