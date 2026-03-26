package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiffNameOnlyHead(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.name", "test")
	run("config", "user.email", "test@test.com")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	run("add", "-A")
	run("commit", "-m", "init")

	// Modify a.txt and add b.txt (staged)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new"), 0644)
	run("add", "-A")

	g := New(dir)
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
