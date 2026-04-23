package ralph

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitpkg "github.com/signalnine/conclave/internal/git"
)

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %s", name, args, out)
	}
}

// installCheckoutBreakingHook installs a post-commit hook that deletes the
// given branch on first fire. Used to simulate a CheckoutBranch failure
// inside BranchFailedWork.
func installCheckoutBreakingHook(t *testing.T, dir, branchToDelete string) {
	t.Helper()
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	script := fmt.Sprintf(`#!/bin/sh
if [ -f "$GIT_DIR/hooks/.fired" ] || [ -f .git/hooks/.fired ]; then exit 0; fi
touch .git/hooks/.fired
git branch -D %s >/dev/null 2>&1 || true
exit 0
`, branchToDelete)
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing hook: %v", err)
	}
}

func TestBranchFailedWork_CreateBranchFailureDoesNotCommitToMain(t *testing.T) {
	dir := t.TempDir()
	runCmd(t, dir, "git", "init", "-q")
	runCmd(t, dir, "git", "config", "user.email", "t@t")
	runCmd(t, dir, "git", "config", "user.name", "t")
	runCmd(t, dir, "git", "checkout", "-qb", "main")
	runCmd(t, dir, "sh", "-c", "echo hi > a.txt")
	runCmd(t, dir, "git", "add", "a.txt")
	runCmd(t, dir, "git", "commit", "-qm", "init")

	// Pre-create a branch that matches what BranchFailedWork will try to create
	taskID := "testtask"
	timestamp := time.Now().Format("20060102-150405")
	collidingBranch := fmt.Sprintf("wip/ralph-fail-%s-%s", taskID, timestamp)
	runCmd(t, dir, "git", "branch", collidingBranch)

	// Make uncommitted changes
	runCmd(t, dir, "sh", "-c", "echo changed > a.txt")

	// Capture main SHA before
	g := gitpkg.New(dir)
	mainBefore, _ := g.RevParse("main")

	state := &State{TaskID: taskID, Iteration: 1, MaxIterations: 5, LastGate: "test", ErrorHash: "deadbeef"}
	if err := BranchFailedWork(g, taskID, state); err != nil {
		t.Fatalf("BranchFailedWork: %v", err)
	}

	mainAfter, _ := g.RevParse("main")
	if mainBefore != mainAfter {
		t.Errorf("main branch SHA advanced unexpectedly: %s -> %s", mainBefore, mainAfter)
	}

	curBranch, _ := g.CurrentBranch()
	if curBranch != "main" {
		t.Errorf("current branch after: %s, want main", curBranch)
	}
}

// TestBranchFailedWork_ReturnsErrorWhenCheckoutBackFails verifies that when
// the final checkout back to the original branch fails (protected-branch
// path), BranchFailedWork returns an error that identifies the failure branch
// the user is now stranded on, instead of returning nil silently.
func TestBranchFailedWork_ReturnsErrorWhenCheckoutBackFails(t *testing.T) {
	dir := t.TempDir()
	runCmd(t, dir, "git", "init", "-q")
	runCmd(t, dir, "git", "config", "user.email", "t@t")
	runCmd(t, dir, "git", "config", "user.name", "t")
	runCmd(t, dir, "git", "checkout", "-qb", "main")
	runCmd(t, dir, "sh", "-c", "echo hi > a.txt")
	runCmd(t, dir, "git", "add", "a.txt")
	runCmd(t, dir, "git", "commit", "-qm", "init")

	// Install a hook that deletes "main" after the failure-branch commit,
	// so CheckoutBranch("main") fails.
	installCheckoutBreakingHook(t, dir, "main")

	// Make uncommitted changes so there's something to commit on the wip branch.
	runCmd(t, dir, "sh", "-c", "echo changed > a.txt")

	g := gitpkg.New(dir)
	state := &State{TaskID: "chktask", Iteration: 1, MaxIterations: 5, LastGate: "test", ErrorHash: "deadbeef"}

	err := BranchFailedWork(g, "chktask", state)
	if err == nil {
		t.Fatalf("expected error when checkout back fails, got nil")
	}
	if !strings.Contains(err.Error(), "wip/ralph-fail-chktask") {
		t.Errorf("error should name the failure branch user is stranded on, got: %v", err)
	}
}

// TestBranchFailedWork_ReturnsErrorWhenCheckoutBackFails_NonProtected covers
// the same error-propagation requirement for the non-protected-branch path.
func TestBranchFailedWork_ReturnsErrorWhenCheckoutBackFails_NonProtected(t *testing.T) {
	dir := t.TempDir()
	runCmd(t, dir, "git", "init", "-q")
	runCmd(t, dir, "git", "config", "user.email", "t@t")
	runCmd(t, dir, "git", "config", "user.name", "t")
	runCmd(t, dir, "git", "checkout", "-qb", "main")
	runCmd(t, dir, "sh", "-c", "echo hi > a.txt")
	runCmd(t, dir, "git", "add", "a.txt")
	runCmd(t, dir, "git", "commit", "-qm", "init")
	// Move to a non-protected branch so BranchFailedWork takes the non-main path.
	runCmd(t, dir, "git", "checkout", "-qb", "feature/x")

	installCheckoutBreakingHook(t, dir, "feature/x")

	runCmd(t, dir, "sh", "-c", "echo changed > a.txt")

	g := gitpkg.New(dir)
	state := &State{TaskID: "feattask", Iteration: 1, MaxIterations: 5, LastGate: "test", ErrorHash: "deadbeef"}

	err := BranchFailedWork(g, "feattask", state)
	if err == nil {
		t.Fatalf("expected error when checkout back fails on non-protected branch, got nil")
	}
}
