package ralph

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GateConfig struct {
	ImplementTimeout int
	TestTimeout      int
	SpecTimeout      int
	QualityTimeout   int
}

func RunTestGate(ctx context.Context, projectDir string, timeout int) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Auto-detect test runner
	var cmd *exec.Cmd
	switch {
	case fileExists(filepath.Join(projectDir, "package.json")):
		cmd = exec.CommandContext(ctx, "npm", "test", "--prefix", projectDir)
	case fileExists(filepath.Join(projectDir, "Cargo.toml")):
		cmd = exec.CommandContext(ctx, "cargo", "test", "--manifest-path", filepath.Join(projectDir, "Cargo.toml"))
	case fileExists(filepath.Join(projectDir, "pyproject.toml")),
		fileExists(filepath.Join(projectDir, "setup.py")):
		cmd = exec.CommandContext(ctx, "python", "-m", "pytest", projectDir)
	case fileExists(filepath.Join(projectDir, "go.mod")):
		cmd = exec.CommandContext(ctx, "go", "test", projectDir+"/...")
	case fileExists(filepath.Join(projectDir, "test.sh")):
		cmd = exec.CommandContext(ctx, filepath.Join(projectDir, "test.sh"))
	default:
		return "WARNING: No test runner detected, skipping test gate", nil
	}
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunSpecGate asks claude to verify spec compliance given the task spec and
// a representation of the current implementation state (typically a git diff
// or the implementer's iteration output). Returns the claude output; callers
// check for "SPEC_PASS" on its own line to confirm compliance.
//
// Prompt passed via stdin to avoid OS arg length limits.
func RunSpecGate(ctx context.Context, spec, currentState string, timeout int) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(`Review this implementation for spec compliance.

## Task Spec
%s

## Current State
%s

## Instructions
Check if the implementation satisfies ALL requirements in the spec.
If fully compliant, output 'SPEC_PASS' on its own line.
Otherwise, list missing, extra, or incorrect items (one per line)
and do NOT output 'SPEC_PASS'.`, spec, currentState)

	cmd := exec.CommandContext(ctx, "claude", "-p", "--output-format", "text")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SpecPassed reports whether the spec gate output indicates compliance.
// It returns true only when SPEC_PASS appears on its own line (ignoring
// surrounding whitespace). A bare substring match would falsely pass when
// the LLM mentions SPEC_PASS in explanatory prose such as
// "I cannot output SPEC_PASS because tests fail."
func SpecPassed(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "SPEC_PASS" {
			return true
		}
	}
	return false
}
