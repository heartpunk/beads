// Package sqlite implements repository management for cross-repo dependencies.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/steveyegge/beads/internal/types"
)

// AddRepository creates a new repository entry
func (s *SQLiteStorage) AddRepository(ctx context.Context, name, description string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repositories (name, description, created_at)
		VALUES (?, ?, ?)
	`, name, description, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}
	return nil
}

// GetRepository retrieves a repository by name
func (s *SQLiteStorage) GetRepository(ctx context.Context, name string) (*types.Repository, error) {
	var repo types.Repository
	var lastSynced sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT name, description, created_at, last_synced
		FROM repositories
		WHERE name = ?
	`, name).Scan(&repo.Name, &repo.Description, &repo.CreatedAt, &lastSynced)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if lastSynced.Valid {
		repo.LastSynced = &lastSynced.String
	}

	return &repo, nil
}

// ListRepositories returns all registered repositories
func (s *SQLiteStorage) ListRepositories(ctx context.Context) ([]*types.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, description, created_at, last_synced
		FROM repositories
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repos []*types.Repository
	for rows.Next() {
		var repo types.Repository
		var lastSynced sql.NullString

		err := rows.Scan(&repo.Name, &repo.Description, &repo.CreatedAt, &lastSynced)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		if lastSynced.Valid {
			repo.LastSynced = &lastSynced.String
		}

		repos = append(repos, &repo)
	}

	return repos, nil
}

// RemoveRepository deletes a repository and all its remotes
func (s *SQLiteStorage) RemoveRepository(ctx context.Context, name string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM repositories WHERE name = ?
	`, name)
	if err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository not found: %s", name)
	}

	return nil
}

// UpdateRepositorySync updates the last_synced timestamp for a repository
func (s *SQLiteStorage) UpdateRepositorySync(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories
		SET last_synced = ?
		WHERE name = ?
	`, time.Now(), name)
	if err != nil {
		return fmt.Errorf("failed to update repository sync time: %w", err)
	}
	return nil
}

// AddRepositoryRemote adds a remote location for a repository
func (s *SQLiteStorage) AddRepositoryRemote(ctx context.Context, repoName, remoteType, remoteValue string, isPrimary bool) error {
	// Validate remote type
	if !types.IsValidRemoteType(remoteType) {
		return fmt.Errorf("invalid remote type: %s", remoteType)
	}

	// Normalize GitHub and GitLab values to shorthand format
	if remoteType == types.RemoteTypeGitHub {
		remoteValue = types.NormalizeGitHubRemote(remoteValue)
	} else if remoteType == types.RemoteTypeGitLab {
		remoteValue = types.NormalizeGitLabRemote(remoteValue)
	}

	// Check if repository exists
	repo, err := s.GetRepository(ctx, repoName)
	if err != nil {
		return fmt.Errorf("failed to check repository: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repository not found: %s", repoName)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If this remote should be primary, unset other primary remotes
	if isPrimary {
		_, err = tx.ExecContext(ctx, `
			UPDATE repository_remotes
			SET is_primary = 0
			WHERE repo_name = ?
		`, repoName)
		if err != nil {
			return fmt.Errorf("failed to unset other primary remotes: %w", err)
		}
	}

	// Insert the remote
	_, err = tx.ExecContext(ctx, `
		INSERT INTO repository_remotes (repo_name, remote_type, remote_value, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, repoName, remoteType, remoteValue, isPrimary, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add repository remote: %w", err)
	}

	return tx.Commit()
}

// GetRepositoryRemotes returns all remotes for a repository
func (s *SQLiteStorage) GetRepositoryRemotes(ctx context.Context, repoName string) ([]*types.RepositoryRemote, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_name, remote_type, remote_value, is_primary, created_at
		FROM repository_remotes
		WHERE repo_name = ?
		ORDER BY is_primary DESC, created_at ASC
	`, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository remotes: %w", err)
	}
	defer rows.Close()

	var remotes []*types.RepositoryRemote
	for rows.Next() {
		var remote types.RepositoryRemote
		var isPrimary int

		err := rows.Scan(&remote.ID, &remote.RepoName, &remote.RemoteType,
			&remote.RemoteValue, &isPrimary, &remote.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan remote: %w", err)
		}

		remote.IsPrimary = isPrimary != 0
		remotes = append(remotes, &remote)
	}

	return remotes, nil
}

// RemoveRepositoryRemote deletes a specific remote
func (s *SQLiteStorage) RemoveRepositoryRemote(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM repository_remotes WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to remove repository remote: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository remote not found: id=%d", id)
	}

	return nil
}

// FindRepositoryByRemote finds a repository that has a matching remote
// Returns the repo name and whether it was found
func (s *SQLiteStorage) FindRepositoryByRemote(ctx context.Context, remoteType, remoteValue string) (string, bool, error) {
	// Normalize the remote value for GitHub and GitLab
	if remoteType == types.RemoteTypeGitHub {
		remoteValue = types.NormalizeGitHubRemote(remoteValue)
	} else if remoteType == types.RemoteTypeGitLab {
		remoteValue = types.NormalizeGitLabRemote(remoteValue)
	}

	var repoName string
	err := s.db.QueryRowContext(ctx, `
		SELECT repo_name
		FROM repository_remotes
		WHERE remote_type = ? AND remote_value = ?
		LIMIT 1
	`, remoteType, remoteValue).Scan(&repoName)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to find repository by remote: %w", err)
	}

	return repoName, true, nil
}

// SetPrimaryRemote marks a remote as primary and unsets others
func (s *SQLiteStorage) SetPrimaryRemote(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the repo name for this remote
	var repoName string
	err = tx.QueryRowContext(ctx, `
		SELECT repo_name FROM repository_remotes WHERE id = ?
	`, id).Scan(&repoName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("repository remote not found: id=%d", id)
	}
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	// Unset all primary remotes for this repo
	_, err = tx.ExecContext(ctx, `
		UPDATE repository_remotes
		SET is_primary = 0
		WHERE repo_name = ?
	`, repoName)
	if err != nil {
		return fmt.Errorf("failed to unset primary remotes: %w", err)
	}

	// Set this remote as primary
	_, err = tx.ExecContext(ctx, `
		UPDATE repository_remotes
		SET is_primary = 1
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to set primary remote: %w", err)
	}

	return tx.Commit()
}
