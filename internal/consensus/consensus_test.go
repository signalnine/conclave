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

func TestTruncateToSentences(t *testing.T) {
	tests := []struct {
		text string
		n    int
		want string
	}{
		{"First. Second. Third. Fourth.", 2, "First. Second."},
		{"No periods here", 3, "No periods here"},
		{"One sentence.", 5, "One sentence."},
		{"Bang! Question? Done.", 2, "Bang! Question?"},
	}
	for _, tt := range tests {
		got := truncateToSentences(tt.text, tt.n)
		if got != tt.want {
			t.Errorf("truncateToSentences(%q, %d) = %q, want %q", tt.text, tt.n, got, tt.want)
		}
	}
}

func TestRunDebateRound(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "Agent A's rebuttal"},
		&mockAgent{name: "B", available: true, response: "Agent B's rebuttal"},
	}

	stage1Results := []AgentResult{
		{Agent: "A", Output: "Full analysis from A about security concerns. The system is vulnerable. Immediate action needed."},
		{Agent: "B", Output: "Full analysis from B about performance issues. Latency is too high. Must optimize."},
	}

	ctx := context.Background()
	rebuttals, err := RunDebateRound(ctx, agents, stage1Results, 60)
	if err != nil {
		t.Fatal(err)
	}

	if len(rebuttals) != 2 {
		t.Fatalf("got %d rebuttals, want 2", len(rebuttals))
	}
	for _, r := range rebuttals {
		if r.Err != nil {
			t.Errorf("agent %s error: %v", r.Agent, r.Err)
		}
		if r.Output == "" {
			t.Errorf("agent %s returned empty rebuttal", r.Agent)
		}
	}
}

func TestRunDebateRoundNeedsMinimumAgents(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "rebuttal"},
	}
	stage1Results := []AgentResult{
		{Agent: "A", Output: "Only one result."},
	}

	ctx := context.Background()
	_, err := RunDebateRound(ctx, agents, stage1Results, 60)
	if err == nil {
		t.Error("expected error with only 1 stage 1 result")
	}
}

func TestRunDebateRoundUnavailableAgent(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "rebuttal A"},
		&mockAgent{name: "B", available: false},
	}

	stage1Results := []AgentResult{
		{Agent: "A", Output: "Analysis from A. Point one. Point two."},
		{Agent: "B", Output: "Analysis from B. Point one. Point two."},
	}

	ctx := context.Background()
	rebuttals, err := RunDebateRound(ctx, agents, stage1Results, 60)
	if err != nil {
		t.Fatal(err)
	}

	if rebuttals[0].Err != nil {
		t.Errorf("agent A should have succeeded: %v", rebuttals[0].Err)
	}
	if rebuttals[1].Err == nil {
		t.Error("agent B should have an error (not available)")
	}
}

func TestRunConsensusWithDebate(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "Analysis A. Security issue found. Needs fix."},
		&mockAgent{name: "B", available: true, response: "Analysis B. Performance concern. Needs optimization."},
	}
	chairmen := []Agent{
		&mockAgent{name: "Chair", available: true, response: "Synthesized consensus output"},
	}

	buildChairman := func(stage1 []AgentResult, rebuttals []AgentResult) string {
		return "synthesize these results"
	}

	ctx := context.Background()
	result, err := RunConsensusWithDebate(ctx, agents, chairmen, "review this code", buildChairman, 60, 60, 60, 1)
	if err != nil {
		t.Fatal(err)
	}

	if result.AgentsSucceeded != 2 {
		t.Errorf("got %d succeeded, want 2", result.AgentsSucceeded)
	}
	if len(result.Stage1Results) != 2 {
		t.Errorf("got %d stage1 results, want 2", len(result.Stage1Results))
	}
	if len(result.Rebuttals) != 2 {
		t.Errorf("got %d rebuttals, want 2", len(result.Rebuttals))
	}
	if result.ChairmanName != "Chair" {
		t.Errorf("chairman = %q, want Chair", result.ChairmanName)
	}
	if result.ChairmanOutput != "Synthesized consensus output" {
		t.Errorf("output = %q", result.ChairmanOutput)
	}
}

func TestRunConsensusWithDebateNoDebateRounds(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: true, response: "Analysis A"},
		&mockAgent{name: "B", available: true, response: "Analysis B"},
	}
	chairmen := []Agent{
		&mockAgent{name: "Chair", available: true, response: "Synthesis"},
	}

	buildChairman := func(stage1 []AgentResult, rebuttals []AgentResult) string {
		return "synthesize"
	}

	ctx := context.Background()
	result, err := RunConsensusWithDebate(ctx, agents, chairmen, "prompt", buildChairman, 60, 60, 60, 0)
	if err != nil {
		t.Fatal(err)
	}

	if result.Rebuttals != nil {
		t.Errorf("expected nil rebuttals with 0 debate rounds, got %d", len(result.Rebuttals))
	}
	if result.ChairmanOutput != "Synthesis" {
		t.Errorf("output = %q", result.ChairmanOutput)
	}
}

// recordingAgent returns different responses on successive Run() calls and
// records the prompts it received. Used to verify that debate round 2
// actually sees round 1 rebuttals instead of re-debating stage 1.
type recordingAgent struct {
	name      string
	responses []string
	prompts   []string
	idx       int
}

func (r *recordingAgent) Name() string    { return r.name }
func (r *recordingAgent) Available() bool { return true }
func (r *recordingAgent) Run(_ context.Context, prompt string) (string, error) {
	r.prompts = append(r.prompts, prompt)
	resp := r.responses[0]
	if r.idx < len(r.responses) {
		resp = r.responses[r.idx]
	}
	r.idx++
	return resp, nil
}

func TestRunConsensusWithDebate_Round2SeesRound1Rebuttals(t *testing.T) {
	// Stage 1 response, then round 1 rebuttal, then round 2 rebuttal
	agentA := &recordingAgent{name: "A", responses: []string{
		"stage1-from-A. Claims X.",
		"rebuttal1-from-A. Disputes B's X claim.",
		"rebuttal2-from-A.",
	}}
	agentB := &recordingAgent{name: "B", responses: []string{
		"stage1-from-B. Claims X is wrong.",
		"rebuttal1-from-B. Defends X.",
		"rebuttal2-from-B.",
	}}

	chairmen := []Agent{
		&mockAgent{name: "Chair", available: true, response: "synthesis"},
	}
	buildChairman := func(_, _ []AgentResult) string { return "prompt" }

	_, err := RunConsensusWithDebate(
		context.Background(),
		[]Agent{agentA, agentB},
		chairmen,
		"initial prompt",
		buildChairman,
		60, 60, 60, 2, // 2 debate rounds
	)
	if err != nil {
		t.Fatal(err)
	}

	// Each agent should have been called 3 times: stage 1, debate round 1, debate round 2
	if len(agentA.prompts) != 3 {
		t.Fatalf("agentA invoked %d times, want 3", len(agentA.prompts))
	}

	// Round 1 (agentA.prompts[1]) should reference stage 1 claims
	if !containsSubstr(agentA.prompts[1], "Claims X") {
		t.Errorf("round 1 prompt should include stage 1 thesis; got: %q", agentA.prompts[1])
	}

	// Round 2 (agentA.prompts[2]) should reference round 1 rebuttals,
	// NOT the original stage 1 theses.
	if !containsSubstr(agentA.prompts[2], "rebuttal1") {
		t.Errorf("round 2 prompt should include round 1 rebuttals; got: %q", agentA.prompts[2])
	}
	if containsSubstr(agentA.prompts[2], "Claims X is wrong") {
		t.Errorf("round 2 prompt should NOT re-include stage 1 theses verbatim; got: %q", agentA.prompts[2])
	}
}

func containsSubstr(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestRunConsensusWithDebateNoAvailableAgents(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "A", available: false},
	}
	chairmen := []Agent{
		&mockAgent{name: "Chair", available: true, response: "Synthesis"},
	}

	buildChairman := func(stage1 []AgentResult, rebuttals []AgentResult) string {
		return "synthesize"
	}

	ctx := context.Background()
	_, err := RunConsensusWithDebate(ctx, agents, chairmen, "prompt", buildChairman, 60, 60, 60, 1)
	if err == nil {
		t.Error("expected error with no available agents")
	}
}
