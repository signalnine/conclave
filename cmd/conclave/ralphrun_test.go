package main

import (
	"testing"
)

func TestRalphRunCmd_EvalFlags(t *testing.T) {
	f := ralphRunCmd.Flags()

	evalFlag := f.Lookup("eval")
	if evalFlag == nil {
		t.Fatal("--eval flag not registered")
	}
	if evalFlag.DefValue != "false" {
		t.Errorf("--eval default = %s, want false", evalFlag.DefValue)
	}

	modelFlag := f.Lookup("eval-model")
	if modelFlag == nil {
		t.Fatal("--eval-model flag not registered")
	}

	timeoutFlag := f.Lookup("eval-timeout")
	if timeoutFlag == nil {
		t.Fatal("--eval-timeout flag not registered")
	}
	if timeoutFlag.DefValue != "120" {
		t.Errorf("--eval-timeout default = %s, want 120", timeoutFlag.DefValue)
	}

	sysPromptFlag := f.Lookup("system-prompt")
	if sysPromptFlag == nil {
		t.Fatal("--system-prompt flag not registered")
	}
}
