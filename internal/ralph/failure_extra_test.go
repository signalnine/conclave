package ralph

import (
	"fmt"
	"os/exec"
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
