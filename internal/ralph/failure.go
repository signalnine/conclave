package ralph

import (
	"fmt"
	"os"
	"time"

	gitpkg "github.com/signalnine/conclave/internal/git"
)

func BranchFailedWork(g *gitpkg.Git, taskID string, state *State) error {
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("wip/ralph-fail-%s-%s", taskID, timestamp)

	currentBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Safety: don't reset protected branches
	if currentBranch == "main" || currentBranch == "master" {
		if err := g.CreateBranch(branchName); err != nil {
			// Branch collision -- retry with unique suffix so we never commit to main.
			branchName = fmt.Sprintf("%s-%d", branchName, time.Now().UnixNano())
			if err := g.CreateBranch(branchName); err != nil {
				return fmt.Errorf("creating failure branch: %w", err)
			}
		}
		if err := g.AddAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: staging failed work: %v\n", err)
		}
		msg := fmt.Sprintf("Ralph Loop failed: %s (on %s)", taskID, currentBranch)
		if err := g.CommitAllowEmpty(msg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: committing to failure branch %s: %v\n", branchName, err)
		}
		if err := g.CheckoutBranch(currentBranch); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: stranded on failure branch %s; checkout %s failed: %v\n", branchName, currentBranch, err)
			return fmt.Errorf("stranded on failure branch %s (checkout %s failed): %w", branchName, currentBranch, err)
		}
		fmt.Fprintf(os.Stderr, "Failed work preserved in branch: %s\n", branchName)
		return nil
	}

	if err := g.CreateBranch(branchName); err != nil {
		// Branch may exist, add timestamp suffix
		branchName = fmt.Sprintf("%s-%d", branchName, time.Now().UnixNano())
		if err := g.CreateBranch(branchName); err != nil {
			return err
		}
	}

	if err := g.AddAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: staging failed work: %v\n", err)
	}
	msg := fmt.Sprintf("Ralph Loop failed: %s\n\nIterations: %d/%d\nLast gate: %s\nError hash: %s",
		taskID, state.Iteration, state.MaxIterations, state.LastGate, state.ErrorHash)
	if err := g.CommitAllowEmpty(msg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: committing to failure branch %s: %v\n", branchName, err)
	}
	g.Push(branchName) // non-fatal if no remote

	if err := g.CheckoutBranch(currentBranch); err != nil {
		return fmt.Errorf("stranded on failure branch %s (checkout %s failed): %w", branchName, currentBranch, err)
	}
	fmt.Fprintf(os.Stderr, "Failed work preserved in branch: %s\n", branchName)
	return nil
}
