package ralph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitState(t *testing.T) {
	dir := t.TempDir()
	s := NewStateManager(dir)
	if err := s.Init("task-1", 5); err != nil {
		t.Fatal(err)
	}
	state, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.TaskID != "task-1" {
		t.Errorf("TaskID = %q", state.TaskID)
	}
	if state.Iteration != 1 {
		t.Errorf("Iteration = %d", state.Iteration)
	}
	if state.MaxIterations != 5 {
		t.Errorf("MaxIterations = %d", state.MaxIterations)
	}
}

func TestUpdateState(t *testing.T) {
	dir := t.TempDir()
	s := NewStateManager(dir)
	s.Init("task-1", 5)

	s.Update("tests", 1, "error output here")
	state, _ := s.Load()
	if state.Iteration != 2 {
		t.Errorf("Iteration = %d, want 2", state.Iteration)
	}
	if state.LastGate != "tests" {
		t.Errorf("LastGate = %q", state.LastGate)
	}
	if state.ErrorHash == "" {
		t.Error("ErrorHash empty")
	}
}

func TestUpdateState_StuckDetection(t *testing.T) {
	dir := t.TempDir()
	s := NewStateManager(dir)
	s.Init("task-1", 10)

	// Same error 3 times should increment stuck count
	for i := 0; i < 3; i++ {
		s.Update("tests", 1, "identical error output")
	}
	state, _ := s.Load()
	if state.StuckCount < 2 {
		t.Errorf("StuckCount = %d, want >= 2", state.StuckCount)
	}
}

func TestCleanup(t *testing.T) {
	dir := t.TempDir()
	s := NewStateManager(dir)
	s.Init("task-1", 5)
	s.Cleanup()

	stateFile := filepath.Join(dir, ".ralph_state.json")
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Error("state file not cleaned up")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	s := NewStateManager(dir)
	if s.Exists() {
		t.Error("should not exist before init")
	}
	s.Init("task-1", 5)
	if !s.Exists() {
		t.Error("should exist after init")
	}
}

func TestAtomicWriteFile_DoesNotLeaveTempOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/target.txt"
	if err := atomicWriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want hello", got)
	}
	// Temp files should be cleaned up.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestAtomicWriteFile_OverwritesInPlace(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/target.txt"
	if err := atomicWriteFile(path, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "second" {
		t.Errorf("got %q, want second", got)
	}
}
