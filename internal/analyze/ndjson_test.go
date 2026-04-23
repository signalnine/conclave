package analyze

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTrace_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	trace, err := ParseTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(trace.ToolCalls))
	}
}

func TestParseTrace_ToolCalls(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")

	// Minimal NDJSON with two assistant messages containing tool_use blocks
	lines := `{"type":"system","subtype":"init","session_id":"test"}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/workspace/src/index.ts","content":"code"}}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"npm test"}}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/workspace/src/__tests__/index.test.ts","content":"test"}}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"git add -A && git commit -m 'feat: init'"}}]}}
{"type":"result","num_turns":4,"duration_ms":60000,"total_cost_usd":1.50}
`
	os.WriteFile(path, []byte(lines), 0644)

	trace, err := ParseTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.ToolCalls) != 4 {
		t.Fatalf("expected 4 tool calls, got %d", len(trace.ToolCalls))
	}
	if trace.ToolCalls[0].Name != "Write" {
		t.Errorf("first call = %q, want Write", trace.ToolCalls[0].Name)
	}
	if trace.ToolCalls[0].FilePath != "/workspace/src/index.ts" {
		t.Errorf("file_path = %q", trace.ToolCalls[0].FilePath)
	}
	if trace.ToolCalls[1].Name != "Bash" {
		t.Errorf("second call = %q, want Bash", trace.ToolCalls[1].Name)
	}
	if trace.ToolCalls[1].Command != "npm test" {
		t.Errorf("command = %q", trace.ToolCalls[1].Command)
	}
	if trace.NumTurns != 4 {
		t.Errorf("num_turns = %d", trace.NumTurns)
	}
	if trace.DurationMS != 60000 {
		t.Errorf("duration_ms = %d", trace.DurationMS)
	}
}

func TestParseTrace_MultipleToolsPerMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")

	// Single assistant message with two tool_use blocks (parallel calls)
	lines := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/workspace/src/a.ts"}},{"type":"tool_use","name":"Read","input":{"file_path":"/workspace/src/b.ts"}}]}}
{"type":"result","num_turns":1,"duration_ms":5000,"total_cost_usd":0.10}
`
	os.WriteFile(path, []byte(lines), 0644)

	trace, err := ParseTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(trace.ToolCalls))
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// True positives -- real test files
		{"/workspace/src/__tests__/foo.test.ts", true},
		{"/workspace/test/foo.test.ts", true},
		{"/workspace/tests/helpers.go", true},
		{"/workspace/spec/parser_spec.rb", true},
		{"/workspace/src/parser_test.go", true},
		{"/workspace/src/parser.test.ts", true},
		{"/workspace/src/parser.spec.js", true},
		{"/workspace/test_parser.py", true},
		// False positives the old substring match got wrong:
		{"/workspace/latest/foo.ts", false},        // "latest" contains "test"
		{"/workspace/src/protest.ts", false},       // "protest" contains "test"
		{"/workspace/src/contestant.go", false},    // "contestant" contains "test"
		{"/workspace/src/specification.ts", false}, // "specification" contains "spec"
		{"/workspace/src/inspector.ts", false},     // "inspector" contains "spec"
		{"/workspace/src/respect.go", false},       // "respect" contains "spec"
		// Plain impl
		{"/workspace/src/foo.ts", false},
		{"/workspace/src/main.go", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsTestFile(tt.path); got != tt.want {
				t.Errorf("IsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsImplFile_NotConfusedByTestSubstrings(t *testing.T) {
	// Files that look like impl but contain test/spec substrings somewhere in the path.
	tests := []struct {
		path string
		want bool
	}{
		{"/workspace/latest/foo.ts", true},        // "latest" was wrongly excluded as test
		{"/workspace/src/protest.ts", true},       // "protest" was wrongly excluded as test
		{"/workspace/src/specification.ts", true}, // "specification" was wrongly excluded
		{"/workspace/src/inspector.ts", true},     // "inspector" was wrongly excluded
		// And actual test files must still be excluded:
		{"/workspace/src/__tests__/foo.test.ts", false},
		{"/workspace/src/foo_test.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsImplFile(tt.path); got != tt.want {
				t.Errorf("IsImplFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseTrace_StringMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")

	// message field as a JSON string (some Claude Code versions do this)
	lines := `{"type":"assistant","message":"{\"content\":[{\"type\":\"tool_use\",\"name\":\"Bash\",\"input\":{\"command\":\"ls\"}}]}"}
{"type":"result","num_turns":1,"duration_ms":1000,"total_cost_usd":0.01}
`
	os.WriteFile(path, []byte(lines), 0644)

	trace, err := ParseTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(trace.ToolCalls))
	}
	if trace.ToolCalls[0].Command != "ls" {
		t.Errorf("command = %q", trace.ToolCalls[0].Command)
	}
}
