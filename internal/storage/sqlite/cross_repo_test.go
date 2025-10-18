package sqlite

import (
	"context"
	"testing"

	"github.com/steveyegge/beads/internal/types"
)

func TestResolveQualifiedID(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	ctx := context.Background()

	// Register a test repository
	if err := store.AddRepository(ctx, "api", "API service"); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}
	if err := store.AddRepositoryRemote(ctx, "api", types.RemoteTypeGitHub, "user/api", true); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	tests := []struct {
		name           string
		qualifiedID    string
		wantRepoName   string
		wantLocalID    string
		wantIsLocal    bool
		wantErr        bool
	}{
		{
			name:        "Local ID",
			qualifiedID: "bd-5",
			wantRepoName: "",
			wantLocalID: "bd-5",
			wantIsLocal: true,
			wantErr: false,
		},
		{
			name:        "Repo name reference",
			qualifiedID: "api:bd-10",
			wantRepoName: "api",
			wantLocalID: "bd-10",
			wantIsLocal: false,
			wantErr: false,
		},
		{
			name:        "GitHub shorthand registered",
			qualifiedID: "gh:user/api:bd-15",
			wantRepoName: "api", // Should resolve to registered repo
			wantLocalID: "bd-15",
			wantIsLocal: false,
			wantErr: false,
		},
		{
			name:        "GitHub shorthand unregistered",
			qualifiedID: "gh:user/unknown:bd-20",
			wantRepoName: "",
			wantLocalID: "gh:user/unknown:bd-20", // Stored as-is
			wantIsLocal: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoName, localID, isLocal, err := store.ResolveQualifiedID(ctx, tt.qualifiedID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveQualifiedID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if repoName != tt.wantRepoName {
				t.Errorf("ResolveQualifiedID() repoName = %v, want %v", repoName, tt.wantRepoName)
			}
			if localID != tt.wantLocalID {
				t.Errorf("ResolveQualifiedID() localID = %v, want %v", localID, tt.wantLocalID)
			}
			if isLocal != tt.wantIsLocal {
				t.Errorf("ResolveQualifiedID() isLocal = %v, want %v", isLocal, tt.wantIsLocal)
			}
		})
	}
}

func TestCrossRepoDependencies(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	ctx := context.Background()

	// Register a remote repository
	if err := store.AddRepository(ctx, "api", "API service"); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}

	// Create a local issue
	localIssue := &types.Issue{
		ID:          "bd-1",
		Title:       "Frontend feature",
		Description: "Needs API support",
		Status:      types.StatusOpen,
		Priority:    1,
		IssueType:   types.TypeTask,
	}
	if err := store.CreateIssue(ctx, localIssue, "test"); err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	// Add cross-repo dependency
	dep := &types.Dependency{
		IssueID:     "bd-1",
		DependsOnID: "api:bd-5",
		Type:        types.DepBlocks,
	}
	if err := store.AddDependency(ctx, dep, "test"); err != nil {
		t.Fatalf("Failed to add cross-repo dependency: %v", err)
	}

	// Verify dependency was stored correctly
	deps, err := store.GetDependencyRecords(ctx, "bd-1")
	if err != nil {
		t.Fatalf("Failed to get dependencies: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].DependsOnID != "api:bd-5" {
		t.Errorf("Expected depends_on_id 'api:bd-5', got '%s'", deps[0].DependsOnID)
	}
	if deps[0].Type != types.DepBlocks {
		t.Errorf("Expected type 'blocks', got '%s'", deps[0].Type)
	}

	// Verify GetCrossRepoDependencies works
	crossDeps, err := store.GetCrossRepoDependencies(ctx)
	if err != nil {
		t.Fatalf("Failed to get cross-repo dependencies: %v", err)
	}

	if len(crossDeps) != 1 {
		t.Fatalf("Expected 1 cross-repo dependency, got %d", len(crossDeps))
	}
}

func TestLocalDependencyCycleDetection(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	ctx := context.Background()

	// Create two local issues
	issue1 := &types.Issue{
		ID:        "bd-1",
		Title:     "Task 1",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}
	issue2 := &types.Issue{
		ID:        "bd-2",
		Title:     "Task 2",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}

	if err := store.CreateIssue(ctx, issue1, "test"); err != nil {
		t.Fatalf("Failed to create issue1: %v", err)
	}
	if err := store.CreateIssue(ctx, issue2, "test"); err != nil {
		t.Fatalf("Failed to create issue2: %v", err)
	}

	// Add bd-1 -> bd-2
	dep1 := &types.Dependency{
		IssueID:     "bd-1",
		DependsOnID: "bd-2",
		Type:        types.DepBlocks,
	}
	if err := store.AddDependency(ctx, dep1, "test"); err != nil {
		t.Fatalf("Failed to add first dependency: %v", err)
	}

	// Try to add bd-2 -> bd-1 (would create cycle)
	dep2 := &types.Dependency{
		IssueID:     "bd-2",
		DependsOnID: "bd-1",
		Type:        types.DepBlocks,
	}
	err := store.AddDependency(ctx, dep2, "test")
	if err == nil {
		t.Fatal("Expected cycle detection error, got nil")
	}

	// Verify error message mentions cycle
	if !contains(err.Error(), "cycle") {
		t.Errorf("Expected error to mention 'cycle', got: %s", err.Error())
	}
}

func TestCrossRepoDependencyBasic(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	ctx := context.Background()

	// Register remote repository
	if err := store.AddRepository(ctx, "api", "API service"); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}

	// Create local issue
	issue1 := &types.Issue{
		ID:        "bd-1",
		Title:     "Frontend task",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}
	if err := store.CreateIssue(ctx, issue1, "test"); err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	// Add cross-repo dependency: bd-1 → api:bd-5
	// (issue_id must be local, depends_on_id can be cross-repo)
	dep1 := &types.Dependency{
		IssueID:     "bd-1",
		DependsOnID: "api:bd-5",
		Type:        types.DepBlocks,
	}
	if err := store.AddDependency(ctx, dep1, "test"); err != nil {
		t.Fatalf("Failed to add cross-repo dependency: %v", err)
	}

	// Verify it was stored correctly
	deps, err := store.GetDependencyRecords(ctx, "bd-1")
	if err != nil {
		t.Fatalf("Failed to get dependencies: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].DependsOnID != "api:bd-5" {
		t.Errorf("Expected depends_on_id 'api:bd-5', got '%s'", deps[0].DependsOnID)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)*2 && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+1] == substr))
}
