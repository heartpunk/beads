package git

import (
	"testing"
)

func TestExtractGitHubRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "HTTPS with .git",
			url:  "https://github.com/user/repo.git",
			want: "user/repo",
		},
		{
			name: "HTTPS without .git",
			url:  "https://github.com/user/repo",
			want: "user/repo",
		},
		{
			name: "SSH format",
			url:  "git@github.com:user/repo.git",
			want: "user/repo",
		},
		{
			name: "SSH without .git",
			url:  "git@github.com:user/repo",
			want: "user/repo",
		},
		{
			name: "Not a GitHub URL",
			url:  "https://gitlab.com/org/proj.git",
			want: "",
		},
		{
			name: "Invalid URL",
			url:  "not-a-url",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGitHubRepo(tt.url)
			if got != tt.want {
				t.Errorf("ExtractGitHubRepo(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractGitLabRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "HTTPS with .git",
			url:  "https://gitlab.com/org/proj.git",
			want: "org/proj",
		},
		{
			name: "HTTPS without .git",
			url:  "https://gitlab.com/org/proj",
			want: "org/proj",
		},
		{
			name: "SSH format",
			url:  "git@gitlab.com:org/proj.git",
			want: "org/proj",
		},
		{
			name: "SSH without .git",
			url:  "git@gitlab.com:org/proj",
			want: "org/proj",
		},
		{
			name: "Not a GitLab URL",
			url:  "https://github.com/user/repo.git",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGitLabRepo(tt.url)
			if got != tt.want {
				t.Errorf("ExtractGitLabRepo(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestAnalyzeRemote(t *testing.T) {
	tests := []struct {
		name       string
		remoteName string
		url        string
		wantType   string
	}{
		{
			name:       "GitHub HTTPS",
			remoteName: "origin",
			url:        "https://github.com/user/repo.git",
			wantType:   "github",
		},
		{
			name:       "GitHub SSH",
			remoteName: "origin",
			url:        "git@github.com:user/repo.git",
			wantType:   "github",
		},
		{
			name:       "GitLab HTTPS",
			remoteName: "origin",
			url:        "https://gitlab.com/org/proj.git",
			wantType:   "gitlab",
		},
		{
			name:       "GitLab SSH",
			remoteName: "origin",
			url:        "git@gitlab.com:org/proj.git",
			wantType:   "gitlab",
		},
		{
			name:       "Other git URL",
			remoteName: "origin",
			url:        "https://bitbucket.org/user/repo.git",
			wantType:   "git-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := AnalyzeRemote(tt.remoteName, tt.url)
			if info.Type != tt.wantType {
				t.Errorf("AnalyzeRemote(%q, %q).Type = %q, want %q",
					tt.remoteName, tt.url, info.Type, tt.wantType)
			}
			if info.Name != tt.remoteName {
				t.Errorf("AnalyzeRemote(%q, %q).Name = %q, want %q",
					tt.remoteName, tt.url, info.Name, tt.remoteName)
			}
			if info.URL != tt.url {
				t.Errorf("AnalyzeRemote(%q, %q).URL = %q, want %q",
					tt.remoteName, tt.url, info.URL, tt.url)
			}
		})
	}
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/user/repo.git", true},
		{"https://github.com/user/repo", true},
		{"git@github.com:user/repo.git", true},
		{"git@github.com:user/repo", true},
		{"https://gitlab.com/org/proj.git", false},
		{"https://bitbucket.org/user/repo.git", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		got := isGitHubURL(tt.url)
		if got != tt.want {
			t.Errorf("isGitHubURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestIsGitLabURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://gitlab.com/org/proj.git", true},
		{"https://gitlab.com/org/proj", true},
		{"git@gitlab.com:org/proj.git", true},
		{"git@gitlab.com:org/proj", true},
		{"https://github.com/user/repo.git", false},
		{"https://bitbucket.org/user/repo.git", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		got := isGitLabURL(tt.url)
		if got != tt.want {
			t.Errorf("isGitLabURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}
