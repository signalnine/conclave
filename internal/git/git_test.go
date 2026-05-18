package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s %v", args, out, err)
		}
	}
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s %v", args, out, err)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	branch, err := g.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("got %q, want main", branch)
	}
}

func TestWorktreeAddAndRemove(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	wtPath := filepath.Join(dir, "wt-test")
	if err := g.WorktreeAdd(wtPath, "test-branch", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtPath); err != nil {
		t.Fatal("worktree not created")
	}
	if err := g.WorktreeRemove(wtPath); err != nil {
		t.Fatal(err)
	}
}

func TestMergeBase(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	run(t, dir, "git", "checkout", "-b", "feature")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feature commit")
	sha, err := g.MergeBase("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Error("empty merge-base")
	}
}

func TestDiff(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)
	run(t, dir, "git", "add", "test.txt")
	run(t, dir, "git", "commit", "-m", "add file")
	diff, err := g.Diff("HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if diff == "" {
		t.Error("empty diff")
	}
}

func TestDiffNameOnly(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	run(t, dir, "git", "add", "a.txt")
	run(t, dir, "git", "commit", "-m", "add a")
	files, err := g.DiffNameOnly("HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "a.txt" {
		t.Errorf("got %v", files)
	}
}

func TestMergeSquash(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	run(t, dir, "git", "checkout", "-b", "feat")
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	run(t, dir, "git", "add", "new.txt")
	run(t, dir, "git", "commit", "-m", "feat commit")
	run(t, dir, "git", "checkout", "main")
	if err := g.MergeSquash("feat"); err != nil {
		t.Fatal(err)
	}
}

func TestHasStagedChanges(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)

	// No staged changes
	if g.HasStagedChanges() {
		t.Error("expected no staged changes")
	}

	// Add a file
	os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged"), 0644)
	run(t, dir, "git", "add", "staged.txt")

	if !g.HasStagedChanges() {
		t.Error("expected staged changes")
	}
}

func TestRevParse(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	sha, err := g.RevParse("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("sha length = %d, want 40", len(sha))
	}
}

func TestDiffNameOnlyHead(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init")

	// Modify a.txt and add b.txt (staged)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new"), 0644)
	run(t, dir, "git", "add", "-A")

	files, err := g.DiffNameOnlyHead()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"a.txt": true, "b.txt": true}
	for _, f := range files {
		if !expected[f] {
			t.Errorf("unexpected file: %s", f)
		}
		delete(expected, f)
	}
	for f := range expected {
		t.Errorf("missing file: %s", f)
	}
}

func TestDiffNameOnlyHead_IncludesUnstaged(t *testing.T) {
	// Regression: DiffNameOnlyHead previously passed --cached, so unstaged
	// working-tree changes (the typical ralph-run state between Claude edits
	// and the final commit) were invisible to the evaluator gate.
	dir := setupTestRepo(t)
	g := New(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init")

	// Modify a.txt (unstaged), create b.txt (untracked but added -> staged),
	// create c.txt (unstaged, untracked).
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new"), 0644)
	run(t, dir, "git", "add", "b.txt")
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("untracked"), 0644)

	files, err := g.DiffNameOnlyHead()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f] = true
	}
	// a.txt (unstaged modification) and b.txt (staged) must both appear.
	if !got["a.txt"] {
		t.Errorf("expected unstaged a.txt in results, got %v", files)
	}
	if !got["b.txt"] {
		t.Errorf("expected staged b.txt in results, got %v", files)
	}
}

func TestDiffHead(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init")

	// Modify a.txt (unstaged) and add b.txt (staged)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new\n"), 0644)
	run(t, dir, "git", "add", "b.txt")

	out, err := g.DiffHead()
	if err != nil {
		t.Fatal(err)
	}
	// Should include both the working-tree change to a.txt and the staged b.txt
	if !contains(out, "a.txt") {
		t.Errorf("expected a.txt in diff, got: %s", out)
	}
	if !contains(out, "b.txt") {
		t.Errorf("expected b.txt in diff, got: %s", out)
	}
	if !contains(out, "changed") {
		t.Errorf("expected 'changed' content in diff, got: %s", out)
	}
}

// TestAddAllExcept_SkipsRalphStateFiles is the regression test for the
// silent-success bug in ralph-run: `git add -A` was sweeping up ralph's
// own state files (.ralph_state.json, .ralph_context.md, .ralph.lock),
// making HasStagedChanges() return true even when claude made no changes
// to the actual project. The "successful" commit then contained only
// ralph noise, masking the failure to parallel-run upstream.
func TestAddAllExcept_SkipsRalphStateFiles(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)

	// Simulate ralph-run leaving its state files behind with no real
	// project changes.
	for _, name := range []string{".ralph_state.json", ".ralph_context.md", ".ralph.lock", ".ralph_raw_1.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("ralph state"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := g.AddAllExcept(".ralph.lock", ".ralph_state.json", ".ralph_context.md", ".ralph_raw_*.txt"); err != nil {
		t.Fatalf("AddAllExcept: %v", err)
	}

	if g.HasStagedChanges() {
		t.Error("expected no staged changes (only ralph state files exist), but HasStagedChanges returned true")
	}

	// Now add a real project file alongside the noise — only the real
	// file should be staged.
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := g.AddAllExcept(".ralph.lock", ".ralph_state.json", ".ralph_context.md", ".ralph_raw_*.txt"); err != nil {
		t.Fatalf("AddAllExcept (with real file): %v", err)
	}
	if !g.HasStagedChanges() {
		t.Error("expected staged changes for real.txt, got none")
	}
	staged, err := g.run("diff", "--cached", "--name-only")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(staged, "real.txt") {
		t.Errorf("expected real.txt in staged set, got: %q", staged)
	}
	for _, name := range []string{".ralph_state.json", ".ralph_context.md", ".ralph.lock", ".ralph_raw_1.txt"} {
		if contains(staged, name) {
			t.Errorf("expected %s NOT to be staged, but staged set was: %q", name, staged)
		}
	}
}

func TestRevList_EmptyWhenBranchHasNoNewCommits(t *testing.T) {
	dir := setupTestRepo(t)
	g := New(dir)
	// Create a branch at HEAD with no new commits.
	run(t, dir, "git", "branch", "feature")
	out, err := g.RevList("HEAD..feature")
	if err != nil {
		t.Fatalf("RevList: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty rev-list for branch with no new commits, got: %q", out)
	}

	// Add a commit on the branch.
	run(t, dir, "git", "checkout", "feature")
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "x.txt")
	run(t, dir, "git", "commit", "-m", "add x")
	run(t, dir, "git", "checkout", "main")

	out, err = g.RevList("HEAD..feature")
	if err != nil {
		t.Fatalf("RevList: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty rev-list for branch with new commit, got empty")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
