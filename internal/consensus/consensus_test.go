package consensus

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type mockAgent struct {
	name      string
	available bool
	response  string
	err       error
	delay     time.Duration
}

func (m *mockAgent) Name() string    { return m.name }
func (m *mockAgent) Available() bool { return m.available }
func (m *mockAgent) Run(ctx context.Context, prompt string) (string, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.response, m.err
}

func TestRunStage1_AllSucceed(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "resp-A"},
		&mockAgent{name: "B", available: true, response: "resp-B"},
		&mockAgent{name: "C", available: true, response: "resp-C"},
	}
	results := RunStage1(context.Background(), agents)
	succeeded := 0
	for _, r := range results {
		if r.Err == nil {
			succeeded++
		}
	}
	if succeeded != 3 {
		t.Errorf("got %d succeeded, want 3", succeeded)
	}
}

func TestRunStage1_OneFails(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "resp-A"},
		&mockAgent{name: "B", available: true, err: fmt.Errorf("API error")},
		&mockAgent{name: "C", available: true, response: "resp-C"},
	}
	results := RunStage1(context.Background(), agents)
	succeeded := 0
	for _, r := range results {
		if r.Err == nil {
			succeeded++
		}
	}
	if succeeded != 2 {
		t.Errorf("got %d succeeded, want 2", succeeded)
	}
}

func TestRunStage1_Timeout(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "fast", delay: 10 * time.Millisecond},
		&mockAgent{name: "B", available: true, response: "slow", delay: 5 * time.Second},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	results := RunStage1(ctx, agents)
	if results[0].Err != nil {
		t.Error("agent A should have succeeded")
	}
	if results[1].Err == nil {
		t.Error("agent B should have timed out")
	}
}

func TestRunStage2_FirstChairmanSucceeds(t *testing.T) {
	chairmen := []Agent{
		&mockAgent{name: "Chair", available: true, response: "synthesis"},
	}
	result, err := RunStage2(context.Background(), chairmen, "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Agent != "Chair" {
		t.Errorf("chairman = %q", result.Agent)
	}
	if result.Output != "synthesis" {
		t.Errorf("output = %q", result.Output)
	}
}

func TestRunStage2_FallbackOnFailure(t *testing.T) {
	chairmen := []Agent{
		&mockAgent{name: "Primary", available: true, err: fmt.Errorf("fail")},
		&mockAgent{name: "Fallback", available: true, response: "synthesis"},
	}
	result, err := RunStage2(context.Background(), chairmen, "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Agent != "Fallback" {
		t.Errorf("chairman = %q, want Fallback", result.Agent)
	}
}

func TestRunStage2_AllFail(t *testing.T) {
	chairmen := []Agent{
		&mockAgent{name: "A", available: true, err: fmt.Errorf("fail")},
		&mockAgent{name: "B", available: true, err: fmt.Errorf("fail")},
	}
	_, err := RunStage2(context.Background(), chairmen, "prompt")
	if err == nil {
		t.Error("expected error when all chairmen fail")
	}
}

func TestRunConsensus_MinOneAgent(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: false},
		&mockAgent{name: "B", available: false},
		&mockAgent{name: "C", available: false},
	}
	_, err := RunConsensus(context.Background(), agents, agents, "prompt", 60, 60)
	if err == nil {
		t.Error("expected error with no available agents")
	}
}
