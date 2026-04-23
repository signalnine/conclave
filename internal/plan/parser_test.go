package plan

import (
	"strings"
	"testing"
	"time"
)

func TestParsePlan_SingleTask(t *testing.T) {
	input := "## Task 1: Create Auth Module\n**Files:**\n- Create: `src/auth.go`\n**Dependencies:** None\n\nImplementation details here.\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}
	if tasks[0].ID != 1 {
		t.Errorf("ID = %d", tasks[0].ID)
	}
	if tasks[0].Title != "Create Auth Module" {
		t.Errorf("Title = %q", tasks[0].Title)
	}
	if len(tasks[0].FilePaths) != 1 || tasks[0].FilePaths[0] != "src/auth.go" {
		t.Errorf("FilePaths = %v", tasks[0].FilePaths)
	}
	if len(tasks[0].DependsOn) != 0 {
		t.Errorf("DependsOn = %v", tasks[0].DependsOn)
	}
}

func TestParsePlan_MultipleTasks(t *testing.T) {
	input := "## Task 1: Setup\n**Dependencies:** None\n\n## Task 2: Auth\n**Dependencies:** Task 1\n\n## Task 3: API\n**Dependencies:** Task 1, Task 2\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks", len(tasks))
	}
	if len(tasks[1].DependsOn) != 1 || tasks[1].DependsOn[0] != 1 {
		t.Errorf("task 2 deps = %v", tasks[1].DependsOn)
	}
	if len(tasks[2].DependsOn) != 2 {
		t.Errorf("task 3 deps = %v", tasks[2].DependsOn)
	}
}

func TestParsePlan_H3Headers(t *testing.T) {
	input := "### Task 1: Works With H3\n**Dependencies:** None\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks", len(tasks))
	}
}

func TestParsePlan_MultipleFiles(t *testing.T) {
	input := "## Task 1: Multi File\n**Files:**\n- Create: `src/a.go`\n- Modify: `src/b.go:10-20`\n- Test: `test/a_test.go`\n**Dependencies:** None\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks[0].FilePaths) != 3 {
		t.Errorf("got %d files: %v", len(tasks[0].FilePaths), tasks[0].FilePaths)
	}
	// Line range suffix should be stripped
	if tasks[0].FilePaths[1] != "src/b.go" {
		t.Errorf("file[1] = %q, want src/b.go", tasks[0].FilePaths[1])
	}
}

func TestParsePlan_Empty(t *testing.T) {
	tasks, err := ParsePlan(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("got %d tasks from empty input", len(tasks))
	}
}

func TestParsePlan_ExtractTaskSpec(t *testing.T) {
	input := "## Task 1: First\nSome content for task 1.\nMore content.\n\n## Task 2: Second\nContent for task 2.\n"
	tasks, _ := ParsePlan(strings.NewReader(input))
	if !strings.Contains(tasks[0].Description, "Some content for task 1") {
		t.Errorf("task 1 description missing content: %q", tasks[0].Description)
	}
	if strings.Contains(tasks[0].Description, "Content for task 2") {
		t.Error("task 1 description contains task 2 content")
	}
}

func TestComputeWaves(t *testing.T) {
	tasks := []Task{
		{ID: 1, DependsOn: nil},
		{ID: 2, DependsOn: []int{1}},
		{ID: 3, DependsOn: nil},
		{ID: 4, DependsOn: []int{2, 3}},
	}
	waves := ComputeWaves(tasks)
	if waves[1] != 0 || waves[3] != 0 {
		t.Errorf("wave[1]=%d, wave[3]=%d, want 0", waves[1], waves[3])
	}
	if waves[2] != 1 {
		t.Errorf("wave[2]=%d, want 1", waves[2])
	}
	if waves[4] != 2 {
		t.Errorf("wave[4]=%d, want 2", waves[4])
	}
}

func TestDetectCycle(t *testing.T) {
	tasks := []Task{
		{ID: 1, DependsOn: []int{2}},
		{ID: 2, DependsOn: []int{1}},
	}
	err := Validate(tasks)
	if err == nil {
		t.Error("expected cycle error")
	}
}

func TestValidate_MissingDep(t *testing.T) {
	tasks := []Task{
		{ID: 1, DependsOn: []int{99}},
	}
	err := Validate(tasks)
	if err == nil {
		t.Error("expected missing dependency error")
	}
}

func TestComputeWaves_CycleDoesNotStackOverflow(t *testing.T) {
	// Regression: ComputeWaves previously stack-overflowed on cycles.
	// Callers should Validate first, but the library must not crash.
	tasks := []Task{
		{ID: 1, DependsOn: []int{2}},
		{ID: 2, DependsOn: []int{1}},
	}
	done := make(chan struct{})
	go func() {
		ComputeWaves(tasks)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ComputeWaves did not terminate on cyclic input")
	}
}

func TestWaveCount_EmptyReturnsZero(t *testing.T) {
	if got := WaveCount(map[int]int{}); got != 0 {
		t.Errorf("WaveCount({}) = %d, want 0", got)
	}
	if got := WaveCount(nil); got != 0 {
		t.Errorf("WaveCount(nil) = %d, want 0", got)
	}
}

func TestParsePlan_TrailingColonDoesNotPanic(t *testing.T) {
	// Regression: parser previously panicked with index-out-of-range on
	// file paths ending with ':' because path[idx+1] indexed past the end.
	input := "## Task 1: Trailing Colon\n**Files:**\n- Create: `foo.go:`\n**Dependencies:** None\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || len(tasks[0].FilePaths) != 1 {
		t.Fatalf("got %d tasks / %d paths", len(tasks), len(tasks[0].FilePaths))
	}
	// Trailing colon is not a line range, so the path should be preserved.
	if tasks[0].FilePaths[0] != "foo.go:" {
		t.Errorf("FilePaths[0] = %q, want foo.go:", tasks[0].FilePaths[0])
	}
}

func TestParsePlan_NonNumericSuffixPreserved(t *testing.T) {
	// Regression: parser only inspected the byte immediately after ':' so
	// a path like 'foo.go:1abc' was incorrectly truncated to 'foo.go'.
	// Only legitimate line-range suffixes (digits, optionally with '-N')
	// should be stripped.
	input := "## Task 1: Odd Suffix\n**Files:**\n- Create: `foo.go:1abc`\n**Dependencies:** None\n"
	tasks, err := ParsePlan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if tasks[0].FilePaths[0] != "foo.go:1abc" {
		t.Errorf("FilePaths[0] = %q, want foo.go:1abc", tasks[0].FilePaths[0])
	}
}

func TestDetectFileOverlaps_CanInduceCycle(t *testing.T) {
	// Regression: DetectFileOverlaps adds tasks[i].ID as dep of tasks[j]
	// for i<j. When task 5 comes before task 3 with file overlap and task 5
	// already depends on task 3, a cycle is introduced. Post-overlap
	// validation must catch it.
	tasks := []Task{
		{ID: 5, FilePaths: []string{"shared.go"}, DependsOn: []int{3}},
		{ID: 3, FilePaths: []string{"shared.go"}},
	}
	tasks = DetectFileOverlaps(tasks)
	if err := Validate(tasks); err == nil {
		t.Fatal("expected cycle error after overlap injection, got nil")
	}
}
