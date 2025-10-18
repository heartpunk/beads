package types

import (
	"testing"
)

func TestParseQualifiedID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *QualifiedID
		wantErr     bool
		errContains string
	}{
		{
			name:  "local issue ID",
			input: "bd-5",
			want: &QualifiedID{
				IssueID: "bd-5",
				IsLocal: true,
			},
		},
		{
			name:  "repository name format",
			input: "api:bd-42",
			want: &QualifiedID{
				RepoName: "api",
				IssueID:  "bd-42",
				IsLocal:  false,
			},
		},
		{
			name:  "GitHub shorthand",
			input: "gh:user/repo:bd-10",
			want: &QualifiedID{
				IssueID:     "bd-10",
				RemoteType:  "github",
				RemoteValue: "user/repo",
				IsLocal:     false,
			},
		},
		{
			name:  "GitLab shorthand",
			input: "gl:org/project:issue-5",
			want: &QualifiedID{
				IssueID:     "issue-5",
				RemoteType:  "gitlab",
				RemoteValue: "org/project",
				IsLocal:     false,
			},
		},
		{
			name:  "HTTPS URL",
			input: "https://github.com/user/repo.git:bd-20",
			want: &QualifiedID{
				IssueID:     "bd-20",
				RemoteType:  "git-url",
				RemoteValue: "https://github.com/user/repo.git",
				IsLocal:     false,
			},
		},
		{
			name:  "SSH URL",
			input: "git@github.com:user/repo.git:bd-30",
			want: &QualifiedID{
				IssueID:     "bd-30",
				RemoteType:  "git-url",
				RemoteValue: "git@github.com:user/repo.git",
				IsLocal:     false,
			},
		},
		{
			name:  "SSH URL with nested path",
			input: "git@git.company.com:team/subteam/project.git:ticket-100",
			want: &QualifiedID{
				IssueID:     "ticket-100",
				RemoteType:  "git-url",
				RemoteValue: "git@git.company.com:team/subteam/project.git",
				IsLocal:     false,
			},
		},
		{
			name:  "custom HTTPS URL",
			input: "https://git.example.com/team/project.git:task-5",
			want: &QualifiedID{
				IssueID:     "task-5",
				RemoteType:  "git-url",
				RemoteValue: "https://git.example.com/team/project.git",
				IsLocal:     false,
			},
		},
		{
			name:  "repo name with dashes and underscores",
			input: "backend-api_v2:bd-5",
			want: &QualifiedID{
				RepoName: "backend-api_v2",
				IssueID:  "bd-5",
				IsLocal:  false,
			},
		},
		{
			name:        "empty reference",
			input:       "",
			wantErr:     true,
			errContains: "empty reference",
		},
		{
			name:        "incomplete GitHub shorthand",
			input:       "gh:bd-5",
			wantErr:     true,
			errContains: "incomplete shorthand",
		},
		{
			name:        "incomplete GitLab shorthand",
			input:       "gl:issue-10",
			wantErr:     true,
			errContains: "incomplete shorthand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQualifiedID(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseQualifiedID() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseQualifiedID() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseQualifiedID() unexpected error: %v", err)
				return
			}

			if got.IssueID != tt.want.IssueID {
				t.Errorf("ParseQualifiedID() IssueID = %v, want %v", got.IssueID, tt.want.IssueID)
			}
			if got.RepoName != tt.want.RepoName {
				t.Errorf("ParseQualifiedID() RepoName = %v, want %v", got.RepoName, tt.want.RepoName)
			}
			if got.RemoteType != tt.want.RemoteType {
				t.Errorf("ParseQualifiedID() RemoteType = %v, want %v", got.RemoteType, tt.want.RemoteType)
			}
			if got.RemoteValue != tt.want.RemoteValue {
				t.Errorf("ParseQualifiedID() RemoteValue = %v, want %v", got.RemoteValue, tt.want.RemoteValue)
			}
			if got.IsLocal != tt.want.IsLocal {
				t.Errorf("ParseQualifiedID() IsLocal = %v, want %v", got.IsLocal, tt.want.IsLocal)
			}
		})
	}
}

func TestQualifiedIDString(t *testing.T) {
	tests := []struct {
		name string
		id   *QualifiedID
		want string
	}{
		{
			name: "local issue",
			id: &QualifiedID{
				IssueID: "bd-5",
				IsLocal: true,
			},
			want: "bd-5",
		},
		{
			name: "repo name format",
			id: &QualifiedID{
				RepoName: "api",
				IssueID:  "bd-42",
				IsLocal:  false,
			},
			want: "api:bd-42",
		},
		{
			name: "GitHub shorthand format",
			id: &QualifiedID{
				IssueID:     "bd-10",
				RemoteType:  "github",
				RemoteValue: "user/repo",
				IsLocal:     false,
			},
			want: "github:user/repo:bd-10",
		},
		{
			name: "GitLab shorthand format",
			id: &QualifiedID{
				IssueID:     "issue-5",
				RemoteType:  "gitlab",
				RemoteValue: "org/project",
				IsLocal:     false,
			},
			want: "gitlab:org/project:issue-5",
		},
		{
			name: "git URL format",
			id: &QualifiedID{
				IssueID:     "bd-20",
				RemoteType:  "git-url",
				RemoteValue: "https://github.com/user/repo.git",
				IsLocal:     false,
			},
			want: "git-url:https://github.com/user/repo.git:bd-20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.String()
			if got != tt.want {
				t.Errorf("QualifiedID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidRemoteType(t *testing.T) {
	tests := []struct {
		name       string
		remoteType string
		want       bool
	}{
		{"local-path is valid", RemoteTypeLocalPath, true},
		{"github is valid", RemoteTypeGitHub, true},
		{"gitlab is valid", RemoteTypeGitLab, true},
		{"git-url is valid", RemoteTypeGitURL, true},
		{"invalid type", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidRemoteType(tt.remoteType)
			if got != tt.want {
				t.Errorf("IsValidRemoteType(%q) = %v, want %v", tt.remoteType, got, tt.want)
			}
		})
	}
}

func TestNormalizeGitHubRemote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already shorthand",
			input: "user/repo",
			want:  "user/repo",
		},
		{
			name:  "HTTPS URL",
			input: "https://github.com/user/repo",
			want:  "user/repo",
		},
		{
			name:  "HTTPS URL with .git",
			input: "https://github.com/user/repo.git",
			want:  "user/repo",
		},
		{
			name:  "SSH URL",
			input: "git@github.com:user/repo.git",
			want:  "user/repo",
		},
		{
			name:  "SSH URL without .git",
			input: "git@github.com:user/repo",
			want:  "user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGitHubRemote(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGitHubRemote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeGitLabRemote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already shorthand",
			input: "org/project",
			want:  "org/project",
		},
		{
			name:  "HTTPS URL",
			input: "https://gitlab.com/org/project",
			want:  "org/project",
		},
		{
			name:  "HTTPS URL with .git",
			input: "https://gitlab.com/org/project.git",
			want:  "org/project",
		},
		{
			name:  "SSH URL",
			input: "git@gitlab.com:org/project.git",
			want:  "org/project",
		},
		{
			name:  "SSH URL without .git",
			input: "git@gitlab.com:org/project",
			want:  "org/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGitLabRemote(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGitLabRemote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
