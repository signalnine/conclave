package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStart_OutputsValidJSON(t *testing.T) {
	dir := t.TempDir()
	// Create using-conclave skill
	skillDir := filepath.Join(dir, "skills", "using-conclave")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Using Conclave\nContent here.\n"), 0644)

	output, err := SessionStart(dir)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	hookOutput, ok := result["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("missing hookSpecificOutput")
	}
	if hookOutput["hookEventName"] != "SessionStart" {
		t.Errorf("hookEventName = %v", hookOutput["hookEventName"])
	}
	ctx, ok := hookOutput["additionalContext"].(string)
	if !ok || ctx == "" {
		t.Error("missing additionalContext")
	}
}

func TestSessionStart_LegacyWarning(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "using-conclave")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("content"), 0644)

	// Create legacy dir
	legacyDir := filepath.Join(t.TempDir(), ".config", "conclave", "skills")
	os.MkdirAll(legacyDir, 0755)
	t.Setenv("HOME", filepath.Dir(filepath.Dir(filepath.Dir(legacyDir))))

	output, err := SessionStart(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should contain warning about legacy skills
	var result map[string]any
	json.Unmarshal([]byte(output), &result)
	hookOutput := result["hookSpecificOutput"].(map[string]any)
	ctx := hookOutput["additionalContext"].(string)
	if len(ctx) == 0 {
		t.Error("expected non-empty context")
	}
}
