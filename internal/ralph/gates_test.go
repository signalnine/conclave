package ralph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunSpecGate_InvokesClaudeWithPromptViaStdin verifies the spec gate:
// (1) calls `claude -p --output-format text`, (2) pipes the full prompt
// via stdin (not a CLI arg), and (3) surfaces the claude output verbatim.
// Uses a fake `claude` script on PATH that records its stdin.
func TestRunSpecGate_InvokesClaudeWithPromptViaStdin(t *testing.T) {
	tmp := t.TempDir()

	recordPath := filepath.Join(tmp, "stdin-record.txt")
	fakeScript := `#!/bin/bash
cat - > "` + recordPath + `"
echo "Compliance verified."
echo "SPEC_PASS"
`
	fakeClaude := filepath.Join(tmp, "claude")
	if err := os.WriteFile(fakeClaude, []byte(fakeScript), 0755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+oldPath)

	spec := "Implement greet() returning 'hi'."
	currentState := "diff --git a/greet.go b/greet.go\n+func greet() string { return \"hi\" }\n"

	out, err := RunSpecGate(context.Background(), spec, currentState, 10)
	if err != nil {
		t.Fatalf("RunSpecGate: %v", err)
	}
	if !strings.Contains(out, "SPEC_PASS") {
		t.Errorf("output missing SPEC_PASS: %q", out)
	}

	stdin, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("reading recorded stdin: %v", err)
	}
	got := string(stdin)
	if !strings.Contains(got, spec) {
		t.Errorf("spec not forwarded to claude stdin; got: %q", got)
	}
	if !strings.Contains(got, currentState) {
		t.Errorf("currentState not forwarded to claude stdin; got: %q", got)
	}
	if !strings.Contains(got, "SPEC_PASS") {
		t.Errorf("prompt should instruct emitting SPEC_PASS; got: %q", got)
	}
}
