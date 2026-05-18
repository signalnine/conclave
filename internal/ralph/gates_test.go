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

// TestRunSpecGate_PassesPermissionModeBypassPermissions verifies that
// nested claude is invoked with --permission-mode bypassPermissions.
// Without it, headless claude hangs on permission prompts (no TTY to
// answer them) and ralph mistakes the timeout output for success — the
// root cause of the parallel-run silent-success cascade.
func TestRunSpecGate_PassesPermissionModeBypassPermissions(t *testing.T) {
	tmp := t.TempDir()
	argsPath := filepath.Join(tmp, "args.txt")
	fakeScript := `#!/bin/bash
printf '%s\n' "$@" > "` + argsPath + `"
cat - > /dev/null
echo "SPEC_PASS"
`
	fakeClaude := filepath.Join(tmp, "claude")
	if err := os.WriteFile(fakeClaude, []byte(fakeScript), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := RunSpecGate(context.Background(), "spec", "state", 10); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("reading recorded args: %v", err)
	}
	got := string(args)
	if !strings.Contains(got, "--permission-mode") || !strings.Contains(got, "bypassPermissions") {
		t.Errorf("expected --permission-mode bypassPermissions in claude args, got:\n%s", got)
	}
}

// TestRunSpecGate_NonComplianceOutputSurfaced verifies the gate returns
// claude's non-compliance feedback verbatim without SPEC_PASS so callers
// can detect failure.
func TestRunSpecGate_NonComplianceOutputSurfaced(t *testing.T) {
	tmp := t.TempDir()
	fakeScript := `#!/bin/bash
cat - > /dev/null
echo "- Missing test file"
echo "- greet() not implemented"
`
	fakeClaude := filepath.Join(tmp, "claude")
	if err := os.WriteFile(fakeClaude, []byte(fakeScript), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := RunSpecGate(context.Background(), "spec", "state", 10)
	if err != nil {
		t.Fatalf("RunSpecGate: %v", err)
	}
	if strings.Contains(out, "SPEC_PASS") {
		t.Errorf("should not contain SPEC_PASS on non-compliance; got: %q", out)
	}
	if !strings.Contains(out, "Missing test file") {
		t.Errorf("non-compliance details missing; got: %q", out)
	}
}

func TestSpecPassed(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"bare token", "SPEC_PASS", true},
		{"trailing newline", "SPEC_PASS\n", true},
		{"leading commentary then token line", "Compliance verified.\nSPEC_PASS\n", true},
		{"token with surrounding whitespace on its own line", "analysis...\n   SPEC_PASS   \ntrailing\n", true},
		{"windows line endings", "ok\r\nSPEC_PASS\r\n", true},
		{"negative prose mentioning token", "I cannot output SPEC_PASS because tests fail.", false},
		{"token embedded mid-sentence", "The gate will accept SPEC_PASS on its own line.", false},
		{"token as code fence content", "```\nSPEC_PASS\n```", true},
		{"no token at all", "- Missing test file\n- greet() not implemented\n", false},
		{"empty output", "", false},
		{"token prefixed by bullet", "- SPEC_PASS\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SpecPassed(tt.output); got != tt.want {
				t.Errorf("SpecPassed(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}
