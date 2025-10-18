// Package main implements the bd CLI remote repository management commands.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/git"
	"github.com/steveyegge/beads/internal/types"
)

// gitDetector is a thin wrapper around git package functions
type gitDetector struct{}

func (g *gitDetector) detectIdentity(path string) (*git.RepositoryIdentity, error) {
	return git.DetectIdentity(path)
}

func (g *gitDetector) analyzeRemote(name, url string) *git.RemoteInfo {
	return git.AnalyzeRemote(name, url)
}

func (g *gitDetector) extractGitHubRepo(url string) string {
	return git.ExtractGitHubRepo(url)
}

func (g *gitDetector) extractGitLabRepo(url string) string {
	return git.ExtractGitLabRepo(url)
}

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote repositories for cross-repo dependencies",
	Long: `Manage remote repositories for cross-repo dependencies.

Repositories can have multiple remotes (local paths, GitHub, GitLab, Git URLs).
This allows correlating 'api:bd-5' with 'gh:user/api:bd-5' when they're the same repo.`,
}

var remoteAddCmd = &cobra.Command{
	Use:   "add [name] [path-or-url]",
	Short: "Register a remote repository",
	Long: `Register a remote repository with a short name.

Supports multiple remote types:
  - Local path:     bd remote add api ../backend-api
  - GitHub:         bd remote add api --github user/api-service
  - GitLab:         bd remote add api --gitlab org/backend
  - HTTPS URL:      bd remote add api --url https://github.com/user/repo.git
  - SSH URL:        bd remote add api --url git@github.com:user/repo.git

You can add multiple remotes to the same repository:
  bd remote add api ../backend-api --primary
  bd remote add api --github user/api-service

This allows 'api:bd-5' and 'gh:user/api-service:bd-5' to resolve to the same repo.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		ctx := context.Background()

		// Get flags
		githubValue, _ := cmd.Flags().GetString("github")
		gitlabValue, _ := cmd.Flags().GetString("gitlab")
		urlValue, _ := cmd.Flags().GetString("url")
		description, _ := cmd.Flags().GetString("description")
		isPrimary, _ := cmd.Flags().GetBool("primary")

		// Count how many remote types are specified
		remoteCount := 0
		var remoteType, remoteValue string

		if len(args) == 2 {
			remoteCount++
			remoteType = types.RemoteTypeLocalPath
			// Convert to absolute path
			absPath, err := filepath.Abs(args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
				os.Exit(1)
			}
			remoteValue = absPath
		}

		if githubValue != "" {
			remoteCount++
			remoteType = types.RemoteTypeGitHub
			remoteValue = githubValue
		}

		if gitlabValue != "" {
			remoteCount++
			remoteType = types.RemoteTypeGitLab
			remoteValue = gitlabValue
		}

		if urlValue != "" {
			remoteCount++
			remoteType = types.RemoteTypeGitURL
			remoteValue = urlValue
		}

		if remoteCount == 0 {
			fmt.Fprintf(os.Stderr, "Error: must specify a path, --github, --gitlab, or --url\n")
			os.Exit(1)
		}

		if remoteCount > 1 {
			fmt.Fprintf(os.Stderr, "Error: can only specify one remote type at a time\n")
			os.Exit(1)
		}

		// Check if repository exists, create if not
		repo, err := store.GetRepository(ctx, repoName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if repo == nil {
			// Create repository first
			if err := store.AddRepository(ctx, repoName, description); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating repository: %v\n", err)
				os.Exit(1)
			}
		}

		// Add the remote
		if err := store.AddRepositoryRemote(ctx, repoName, remoteType, remoteValue, isPrimary); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Warn if no GitHub/GitLab remote for local paths
		if remoteType == types.RemoteTypeLocalPath {
			remotes, err := store.GetRepositoryRemotes(ctx, repoName)
			if err == nil {
				hasRemoteRemote := false
				for _, r := range remotes {
					if r.RemoteType == types.RemoteTypeGitHub ||
						r.RemoteType == types.RemoteTypeGitLab ||
						r.RemoteType == types.RemoteTypeGitURL {
						hasRemoteRemote = true
						break
					}
				}
				if !hasRemoteRemote {
					yellow := color.New(color.FgYellow).SprintFunc()
					fmt.Fprintf(os.Stderr, "\n%s Tip: Consider adding a GitHub/GitLab remote too for better correlation:\n", yellow("💡"))
					fmt.Fprintf(os.Stderr, "  bd remote add %s --github user/repo\n\n", repoName)
					fmt.Fprintf(os.Stderr, "This helps identify the same repo across different locations.\n\n")
				}
			}
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":      "added",
				"repo_name":   repoName,
				"remote_type": remoteType,
				"remote_value": remoteValue,
				"is_primary":  isPrimary,
			})
			return
		}

		green := color.New(color.FgGreen).SprintFunc()
		primaryStr := ""
		if isPrimary {
			primaryStr = " (primary)"
		}
		fmt.Printf("%s Added %s remote '%s' → %s%s\n",
			green("✓"), remoteType, repoName, remoteValue, primaryStr)
	},
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all remote repositories",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		repos, err := store.ListRepositories(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			// Include remotes for each repository
			type repoWithRemotes struct {
				types.Repository
				Remotes []*types.RepositoryRemote `json:"remotes"`
			}
			result := make([]repoWithRemotes, 0, len(repos))
			for _, repo := range repos {
				remotes, err := store.GetRepositoryRemotes(ctx, repo.Name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting remotes for %s: %v\n", repo.Name, err)
					os.Exit(1)
				}
				result = append(result, repoWithRemotes{
					Repository: *repo,
					Remotes:    remotes,
				})
			}
			outputJSON(result)
			return
		}

		if len(repos) == 0 {
			fmt.Println("\nNo remote repositories registered")
			fmt.Println("\nUse 'bd remote add <name> <path>' to register a repository")
			return
		}

		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n%s Remote Repositories:\n\n", cyan("🌐"))

		for _, repo := range repos {
			fmt.Printf("  %s\n", repo.Name)
			if repo.Description != "" {
				gray := color.New(color.FgHiBlack).SprintFunc()
				fmt.Printf("    %s\n", gray(repo.Description))
			}

			remotes, err := store.GetRepositoryRemotes(ctx, repo.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
				continue
			}

			for _, remote := range remotes {
				primaryTag := ""
				if remote.IsPrimary {
					yellow := color.New(color.FgYellow).SprintFunc()
					primaryTag = yellow(" [primary]")
				}
				typeColor := color.New(color.FgHiBlack).SprintFunc()
				fmt.Printf("    %s %s%s\n",
					typeColor(fmt.Sprintf("%-10s", remote.RemoteType)),
					remote.RemoteValue,
					primaryTag)
			}
			fmt.Println()
		}
	},
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a remote repository and all its remotes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		ctx := context.Background()

		if err := store.RemoveRepository(ctx, repoName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":    "removed",
				"repo_name": repoName,
			})
			return
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Removed repository '%s' and all its remotes\n", green("✓"), repoName)
	},
}

var remoteSyncCmd = &cobra.Command{
	Use:   "sync [name]",
	Short: "Update last sync timestamp for a repository",
	Long: `Update the last_synced timestamp for a repository.

This is useful for tracking when you last synchronized with a remote repository.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		ctx := context.Background()

		if err := store.UpdateRepositorySync(ctx, repoName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":    "synced",
				"repo_name": repoName,
			})
			return
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Synced repository '%s'\n", green("✓"), repoName)
	},
}

var remoteRemoveRemoteCmd = &cobra.Command{
	Use:   "remove-remote [repo-name] [remote-type] [remote-value]",
	Short: "Remove a specific remote from a repository",
	Long: `Remove a specific remote from a repository while keeping the repo registered.

Examples:
  bd remote remove-remote api local-path /home/user/api
  bd remote remove-remote api github user/api-service`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		remoteType := args[1]
		remoteValue := args[2]

		ctx := context.Background()

		// Get all remotes for this repo
		remotes, err := store.GetRepositoryRemotes(ctx, repoName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Find matching remote
		var targetID int
		found := false
		for _, remote := range remotes {
			if remote.RemoteType == remoteType {
				// Normalize for comparison
				compareValue := remoteValue
				if remoteType == types.RemoteTypeGitHub {
					compareValue = types.NormalizeGitHubRemote(remoteValue)
				} else if remoteType == types.RemoteTypeGitLab {
					compareValue = types.NormalizeGitLabRemote(remoteValue)
				}
				if remote.RemoteValue == compareValue || remote.RemoteValue == remoteValue {
					targetID = remote.ID
					found = true
					break
				}
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "Error: remote not found: %s %s for repository %s\n",
				remoteType, remoteValue, repoName)
			os.Exit(1)
		}

		if err := store.RemoveRepositoryRemote(ctx, targetID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":       "removed",
				"repo_name":    repoName,
				"remote_type":  remoteType,
				"remote_value": remoteValue,
			})
			return
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Removed %s remote '%s' from %s\n",
			green("✓"), remoteType, remoteValue, repoName)
	},
}

var remoteAutoDetectCmd = &cobra.Command{
	Use:   "auto-detect [path]",
	Short: "Auto-detect and register a repository from a directory",
	Long: `Auto-detect repository information from a directory and register it.

This command:
- Checks if the directory is a git repository
- Checks if it has a .beads directory
- Extracts git remotes (GitHub, GitLab, etc.)
- Suggests a repository name
- Registers the repository with all detected remotes

If no path is provided, uses the current directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		ctx := context.Background()

		// Import the git package
		gitPkg := &gitDetector{}
		identity, err := gitPkg.detectIdentity(targetPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting repository: %v\n", err)
			os.Exit(1)
		}

		// Must have .beads directory
		if !identity.HasBeads {
			fmt.Fprintf(os.Stderr, "Error: %s does not contain a .beads directory\n", identity.Path)
			fmt.Fprintf(os.Stderr, "Run 'bd init' first to initialize beads in this repository.\n")
			os.Exit(1)
		}

		// Get suggested name (or allow override)
		repoName, _ := cmd.Flags().GetString("name")
		if repoName == "" {
			repoName = identity.SuggestedName
		}

		description, _ := cmd.Flags().GetString("description")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if dryRun || !jsonOutput {
			cyan := color.New(color.FgCyan).SprintFunc()
			fmt.Printf("\n%s Repository Detection:\n\n", cyan("🔍"))
			fmt.Printf("  Path: %s\n", identity.Path)
			fmt.Printf("  Suggested name: %s\n", repoName)
			if identity.IsGitRepo {
				fmt.Printf("  Git repository: Yes\n")
				if len(identity.Remotes) > 0 {
					fmt.Printf("  Git remotes: %d\n\n", len(identity.Remotes))
					for name, url := range identity.Remotes {
						info := gitPkg.analyzeRemote(name, url)
						typeColor := color.New(color.FgHiBlack).SprintFunc()
						fmt.Printf("    %s %s → %s\n",
							typeColor(fmt.Sprintf("%-10s", info.Type)),
							name,
							url)
					}
				}
			} else {
				fmt.Printf("  Git repository: No\n")
			}
			fmt.Println()
		}

		if dryRun {
			fmt.Println("Dry run - no changes made")
			return
		}

		// Register the repository
		if err := store.AddRepository(ctx, repoName, description); err != nil {
			// Check if it already exists
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				yellow := color.New(color.FgYellow).SprintFunc()
				fmt.Fprintf(os.Stderr, "%s Repository '%s' already registered\n", yellow("⚠"), repoName)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

		// Add local path remote first (as primary)
		if err := store.AddRepositoryRemote(ctx, repoName, types.RemoteTypeLocalPath, identity.Path, true); err != nil {
			if !strings.Contains(err.Error(), "UNIQUE constraint") {
				fmt.Fprintf(os.Stderr, "Error adding local path: %v\n", err)
				os.Exit(1)
			}
		}

		// Add git remotes
		remoteCount := 1 // local path
		for _, info := range identity.RemoteInfos {
			var remoteType, remoteValue string

			if info.Type == "github" {
				remoteType = types.RemoteTypeGitHub
				remoteValue = gitPkg.extractGitHubRepo(info.URL)
			} else if info.Type == "gitlab" {
				remoteType = types.RemoteTypeGitLab
				remoteValue = gitPkg.extractGitLabRepo(info.URL)
			} else {
				remoteType = types.RemoteTypeGitURL
				remoteValue = info.URL
			}

			if remoteValue == "" {
				continue
			}

			if err := store.AddRepositoryRemote(ctx, repoName, remoteType, remoteValue, false); err != nil {
				if !strings.Contains(err.Error(), "UNIQUE constraint") {
					fmt.Fprintf(os.Stderr, "Warning: failed to add %s remote: %v\n", remoteType, err)
				}
			} else {
				remoteCount++
			}
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":       "registered",
				"repo_name":    repoName,
				"path":         identity.Path,
				"remote_count": remoteCount,
			})
			return
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Registered repository '%s' with %d remote(s)\n",
			green("✓"), repoName, remoteCount)
	},
}

var remoteCorrelateCmd = &cobra.Command{
	Use:   "correlate",
	Short: "Find and link repositories that refer to the same codebase",
	Long: `Analyze registered repositories and identify which ones refer to the same codebase.

This command looks for repositories with matching GitHub/GitLab remotes and suggests
consolidating them under a single repository name with multiple remotes.

This helps resolve cases where the same repo is registered multiple times with different names.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		repos, err := store.ListRepositories(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Build a map of remote value -> repo names
		type remoteMatch struct {
			RemoteType  string
			RemoteValue string
			RepoNames   []string
		}

		remoteMap := make(map[string]*remoteMatch)

		for _, repo := range repos {
			remotes, err := store.GetRepositoryRemotes(ctx, repo.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting remotes for %s: %v\n", repo.Name, err)
				continue
			}

			for _, remote := range remotes {
				// Only consider GitHub, GitLab, and git-url for correlation
				if remote.RemoteType == types.RemoteTypeLocalPath {
					continue
				}

				key := remote.RemoteType + ":" + remote.RemoteValue
				if match, exists := remoteMap[key]; exists {
					match.RepoNames = append(match.RepoNames, repo.Name)
				} else {
					remoteMap[key] = &remoteMatch{
						RemoteType:  remote.RemoteType,
						RemoteValue: remote.RemoteValue,
						RepoNames:   []string{repo.Name},
					}
				}
			}
		}

		// Find duplicates
		var duplicates []*remoteMatch
		for _, match := range remoteMap {
			if len(match.RepoNames) > 1 {
				duplicates = append(duplicates, match)
			}
		}

		if len(duplicates) == 0 {
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"correlations": []interface{}{},
				})
			} else {
				green := color.New(color.FgGreen).SprintFunc()
				fmt.Printf("%s No duplicate repositories found\n", green("✓"))
				fmt.Println("\nAll repositories have unique remotes.")
			}
			return
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"correlations": duplicates,
			})
			return
		}

		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\n%s Found %d correlation(s):\n\n", yellow("⚠"), len(duplicates))

		for i, dup := range duplicates {
			fmt.Printf("%d. %s: %s\n", i+1, dup.RemoteType, dup.RemoteValue)
			fmt.Printf("   Registered as: %s\n", strings.Join(dup.RepoNames, ", "))
			fmt.Println()
		}

		fmt.Println("Recommendation: Consolidate these by using one repository name with")
		fmt.Println("multiple remotes. You can remove duplicates with 'bd remote remove <name>'")
		fmt.Println("and add additional remotes with 'bd remote add <name> --github user/repo'")
	},
}

func init() {
	// Flags for remote add
	remoteAddCmd.Flags().String("github", "", "GitHub repository (user/repo format)")
	remoteAddCmd.Flags().String("gitlab", "", "GitLab repository (org/proj format)")
	remoteAddCmd.Flags().String("url", "", "Git URL (HTTPS or SSH)")
	remoteAddCmd.Flags().String("description", "", "Repository description")
	remoteAddCmd.Flags().Bool("primary", false, "Mark this as the primary remote")

	// Flags for remote auto-detect
	remoteAutoDetectCmd.Flags().String("name", "", "Override suggested repository name")
	remoteAutoDetectCmd.Flags().String("description", "", "Repository description")
	remoteAutoDetectCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")

	// Add subcommands
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteSyncCmd)
	remoteCmd.AddCommand(remoteRemoveRemoteCmd)
	remoteCmd.AddCommand(remoteAutoDetectCmd)
	remoteCmd.AddCommand(remoteCorrelateCmd)

	// Register with root
	rootCmd.AddCommand(remoteCmd)
}
