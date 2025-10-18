package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/types"
)

// TestAddRepository tests adding a new repository
func TestAddRepository(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API service")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Verify repository was added
	repo, err := store.GetRepository(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("Expected repository to exist, got nil")
	}

	if repo.Name != "api" {
		t.Errorf("Expected name 'api', got %s", repo.Name)
	}

	if repo.Description != "Backend API service" {
		t.Errorf("Expected description 'Backend API service', got %s", repo.Description)
	}
}

// TestGetRepositoryNotFound tests getting a non-existent repository
func TestGetRepositoryNotFound(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	repo, err := store.GetRepository(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo != nil {
		t.Error("Expected nil for non-existent repository, got result")
	}
}

// TestListRepositories tests listing all repositories
func TestListRepositories(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add multiple repositories
	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	err = store.AddRepository(ctx, "frontend", "Web Frontend")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	err = store.AddRepository(ctx, "mobile", "Mobile App")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// List repositories
	repos, err := store.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("Expected 3 repositories, got %d", len(repos))
	}

	// Verify alphabetical order
	if len(repos) >= 3 {
		if repos[0].Name != "api" {
			t.Errorf("Expected first repo to be 'api', got %s", repos[0].Name)
		}
		if repos[1].Name != "frontend" {
			t.Errorf("Expected second repo to be 'frontend', got %s", repos[1].Name)
		}
		if repos[2].Name != "mobile" {
			t.Errorf("Expected third repo to be 'mobile', got %s", repos[2].Name)
		}
	}
}

// TestRemoveRepository tests removing a repository
func TestRemoveRepository(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add repository
	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Remove repository
	err = store.RemoveRepository(ctx, "api")
	if err != nil {
		t.Fatalf("RemoveRepository failed: %v", err)
	}

	// Verify repository was removed
	repo, err := store.GetRepository(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo != nil {
		t.Error("Expected repository to be removed, still exists")
	}
}

// TestRemoveRepositoryNotFound tests removing a non-existent repository
func TestRemoveRepositoryNotFound(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.RemoveRepository(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error when removing non-existent repository, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestAddRepositoryRemote tests adding a remote to a repository
func TestAddRepositoryRemote(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create repository
	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add local path remote
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeLocalPath, "/home/user/api", true)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Verify remote was added
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 1 {
		t.Fatalf("Expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].RemoteType != types.RemoteTypeLocalPath {
		t.Errorf("Expected remote type 'local-path', got %s", remotes[0].RemoteType)
	}

	if remotes[0].RemoteValue != "/home/user/api" {
		t.Errorf("Expected remote value '/home/user/api', got %s", remotes[0].RemoteValue)
	}

	if !remotes[0].IsPrimary {
		t.Error("Expected remote to be primary")
	}
}

// TestAddMultipleRemotes tests adding multiple remotes to a repository
func TestAddMultipleRemotes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create repository
	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add local path remote (primary)
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeLocalPath, "/home/user/api", true)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Add GitHub remote
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api-service", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Add GitLab remote
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitLab, "org/backend", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Verify all remotes were added
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 3 {
		t.Fatalf("Expected 3 remotes, got %d", len(remotes))
	}

	// Verify primary remote comes first
	if !remotes[0].IsPrimary {
		t.Error("Expected first remote to be primary")
	}
}

// TestGitHubNormalization tests that GitHub URLs are normalized to shorthand
func TestGitHubNormalization(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add GitHub remote with full HTTPS URL
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "https://github.com/user/repo.git", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Verify it was normalized to shorthand
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 1 {
		t.Fatalf("Expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].RemoteValue != "user/repo" {
		t.Errorf("Expected normalized value 'user/repo', got %s", remotes[0].RemoteValue)
	}
}

// TestGitLabNormalization tests that GitLab URLs are normalized to shorthand
func TestGitLabNormalization(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add GitLab remote with SSH URL
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitLab, "git@gitlab.com:org/project.git", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Verify it was normalized to shorthand
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 1 {
		t.Fatalf("Expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].RemoteValue != "org/project" {
		t.Errorf("Expected normalized value 'org/project', got %s", remotes[0].RemoteValue)
	}
}

// TestSetPrimaryRemote tests changing which remote is primary
func TestSetPrimaryRemote(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add two remotes
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeLocalPath, "/home/user/api", true)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Get remotes
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	// Find the GitHub remote ID
	var githubID int
	for _, r := range remotes {
		if r.RemoteType == types.RemoteTypeGitHub {
			githubID = r.ID
			break
		}
	}

	// Set GitHub remote as primary
	err = store.SetPrimaryRemote(ctx, githubID)
	if err != nil {
		t.Fatalf("SetPrimaryRemote failed: %v", err)
	}

	// Verify primary was switched
	remotes, err = store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	// GitHub remote should be first (primary)
	if remotes[0].RemoteType != types.RemoteTypeGitHub {
		t.Errorf("Expected GitHub remote to be primary, got %s", remotes[0].RemoteType)
	}

	if !remotes[0].IsPrimary {
		t.Error("Expected GitHub remote to be marked as primary")
	}

	// Local path should no longer be primary
	if remotes[1].IsPrimary {
		t.Error("Expected local-path remote to no longer be primary")
	}
}

// TestFindRepositoryByRemote tests finding a repository by its remote
func TestFindRepositoryByRemote(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create repository with GitHub remote
	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api-service", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Find repository by GitHub remote
	repoName, found, err := store.FindRepositoryByRemote(ctx, types.RemoteTypeGitHub, "user/api-service")
	if err != nil {
		t.Fatalf("FindRepositoryByRemote failed: %v", err)
	}

	if !found {
		t.Fatal("Expected to find repository, got not found")
	}

	if repoName != "api" {
		t.Errorf("Expected repo name 'api', got %s", repoName)
	}
}

// TestFindRepositoryByRemoteNormalization tests finding with non-normalized values
func TestFindRepositoryByRemoteNormalization(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add GitHub remote (will be normalized to "user/repo")
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/repo", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Search with full HTTPS URL (should normalize and match)
	repoName, found, err := store.FindRepositoryByRemote(ctx, types.RemoteTypeGitHub, "https://github.com/user/repo.git")
	if err != nil {
		t.Fatalf("FindRepositoryByRemote failed: %v", err)
	}

	if !found {
		t.Fatal("Expected to find repository with normalized URL")
	}

	if repoName != "api" {
		t.Errorf("Expected repo name 'api', got %s", repoName)
	}
}

// TestRemoveRepositoryRemote tests removing a specific remote
func TestRemoveRepositoryRemote(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add two remotes
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeLocalPath, "/home/user/api", true)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Get remotes
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 2 {
		t.Fatalf("Expected 2 remotes, got %d", len(remotes))
	}

	// Remove GitHub remote
	var githubID int
	for _, r := range remotes {
		if r.RemoteType == types.RemoteTypeGitHub {
			githubID = r.ID
			break
		}
	}

	err = store.RemoveRepositoryRemote(ctx, githubID)
	if err != nil {
		t.Fatalf("RemoveRepositoryRemote failed: %v", err)
	}

	// Verify only local remote remains
	remotes, err = store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 1 {
		t.Fatalf("Expected 1 remote after removal, got %d", len(remotes))
	}

	if remotes[0].RemoteType != types.RemoteTypeLocalPath {
		t.Errorf("Expected remaining remote to be local-path, got %s", remotes[0].RemoteType)
	}
}

// TestRemoveRepositoryCascadesRemotes tests that removing a repository cascades to remotes
func TestRemoveRepositoryCascadesRemotes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Add multiple remotes
	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeLocalPath, "/home/user/api", true)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	err = store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api", false)
	if err != nil {
		t.Fatalf("AddRepositoryRemote failed: %v", err)
	}

	// Remove repository
	err = store.RemoveRepository(ctx, "api")
	if err != nil {
		t.Fatalf("RemoveRepository failed: %v", err)
	}

	// Verify remotes were cascaded
	remotes, err := store.GetRepositoryRemotes(ctx, "api")
	if err != nil {
		t.Fatalf("GetRepositoryRemotes failed: %v", err)
	}

	if len(remotes) != 0 {
		t.Errorf("Expected 0 remotes after repository removal, got %d", len(remotes))
	}
}

// TestAddRepositoryRemoteInvalidType tests validation of remote types
func TestAddRepositoryRemoteInvalidType(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepository(ctx, "api", "Backend API")
	if err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	err = store.AddRepositoryRemote(ctx, "api", "invalid-type", "/path", false)
	if err == nil {
		t.Fatal("Expected error for invalid remote type, got nil")
	}

	if !strings.Contains(err.Error(), "invalid remote type") {
		t.Errorf("Expected 'invalid remote type' error, got: %v", err)
	}
}

// TestAddRepositoryRemoteNonExistentRepo tests adding remote to non-existent repository
func TestAddRepositoryRemoteNonExistentRepo(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := store.AddRepositoryRemote(ctx, "nonexistent", types.RemoteTypeLocalPath, "/path", false)
	if err == nil {
		t.Fatal("Expected error for non-existent repository, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}
