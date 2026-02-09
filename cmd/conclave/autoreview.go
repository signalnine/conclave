package main

import (
	"fmt"
	"os"
	"strings"

	gitpkg "github.com/signalnine/conclave/internal/git"
	"github.com/spf13/cobra"
)

var autoReviewCmd = &cobra.Command{
	Use:   "auto-review [description]",
	Short: "Auto-detect SHAs and run consensus code review",
	Long:  "Convenience wrapper that auto-detects base/head SHAs from git history, then runs consensus code review.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAutoReview,
}

func init() {
	autoReviewCmd.Flags().String("base-sha", "", "Override base SHA (default: auto-detect from origin/main)")
	autoReviewCmd.Flags().String("head-sha", "", "Override head SHA (default: HEAD)")
	autoReviewCmd.Flags().String("plan-file", "", "Path to implementation plan file")
	rootCmd.AddCommand(autoReviewCmd)
}

func runAutoReview(cmd *cobra.Command, args []string) error {
	description := strings.Join(args, " ")
	g := gitpkg.New(".")

	baseSHA, _ := cmd.Flags().GetString("base-sha")
	headSHA, _ := cmd.Flags().GetString("head-sha")

	if headSHA == "" {
		var err error
		headSHA, err = g.RevParse("HEAD")
		if err != nil {
			return fmt.Errorf("failed to get HEAD: %w", err)
		}
	}

	if baseSHA == "" {
		// Try origin/main first, fall back to main
		var err error
		baseSHA, err = g.MergeBase("origin/main", headSHA)
		if err != nil {
			baseSHA, err = g.MergeBase("main", headSHA)
			if err != nil {
				baseSHA, err = g.MergeBase("origin/master", headSHA)
				if err != nil {
					baseSHA, err = g.MergeBase("master", headSHA)
					if err != nil {
						return fmt.Errorf("could not determine base SHA: %w", err)
					}
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Auto-review: base=%s head=%s\n", baseSHA[:8], headSHA[:8])

	// Delegate to consensus command
	planFile, _ := cmd.Flags().GetString("plan-file")
	consensusArgs := []string{
		"--mode=code-review",
		"--base-sha=" + baseSHA,
		"--head-sha=" + headSHA,
		"--description=" + description,
	}
	if planFile != "" {
		consensusArgs = append(consensusArgs, "--plan-file="+planFile)
	}

	consensusCmd.SetArgs(consensusArgs)
	return consensusCmd.RunE(consensusCmd, nil)
}
