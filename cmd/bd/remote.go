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
	"github.com/steveyegge/beads/internal/types"
)

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

func init() {
	// Flags for remote add
	remoteAddCmd.Flags().String("github", "", "GitHub repository (user/repo format)")
	remoteAddCmd.Flags().String("gitlab", "", "GitLab repository (org/proj format)")
	remoteAddCmd.Flags().String("url", "", "Git URL (HTTPS or SSH)")
	remoteAddCmd.Flags().String("description", "", "Repository description")
	remoteAddCmd.Flags().Bool("primary", false, "Mark this as the primary remote")

	// Add subcommands
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteSyncCmd)
	remoteCmd.AddCommand(remoteRemoveRemoteCmd)

	// Register with root
	rootCmd.AddCommand(remoteCmd)
}
