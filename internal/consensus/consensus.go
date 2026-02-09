package consensus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type AgentResult struct {
	Agent  string
	Output string
	Err    error
}

type ConsensusResult struct {
	Stage1Results   []AgentResult
	ChairmanName    string
	ChairmanOutput  string
	OutputFile      string
	AgentsSucceeded int
}

func RunStage1(ctx context.Context, agents []Agent) []AgentResult {
	results := make([]AgentResult, len(agents))
	var wg sync.WaitGroup

	for i, agent := range agents {
		wg.Add(1)
		go func(i int, a Agent) {
			defer wg.Done()
			output, err := a.Run(ctx, "")
			results[i] = AgentResult{Agent: a.Name(), Output: output, Err: err}
		}(i, agent)
	}

	wg.Wait()
	return results
}

func runStage1WithPrompt(ctx context.Context, agents []Agent, prompt string) []AgentResult {
	results := make([]AgentResult, len(agents))
	var wg sync.WaitGroup

	for i, agent := range agents {
		wg.Add(1)
		go func(i int, a Agent) {
			defer wg.Done()
			output, err := a.Run(ctx, prompt)
			results[i] = AgentResult{Agent: a.Name(), Output: output, Err: err}
		}(i, agent)
	}

	wg.Wait()
	return results
}

func RunStage2(ctx context.Context, chairmen []Agent, prompt string) (AgentResult, error) {
	for _, chairman := range chairmen {
		if !chairman.Available() {
			continue
		}
		output, err := chairman.Run(ctx, prompt)
		if err == nil && output != "" {
			return AgentResult{Agent: chairman.Name(), Output: output}, nil
		}
		fmt.Fprintf(os.Stderr, "  %s: FAILED (%v)\n", chairman.Name(), err)
	}
	return AgentResult{}, fmt.Errorf("all chairman agents failed")
}

func RunConsensus(ctx context.Context, agents, chairmen []Agent, prompt string, stage1Timeout, stage2Timeout int) (*ConsensusResult, error) {
	// Filter available agents
	var available []Agent
	for _, a := range agents {
		if a.Available() {
			available = append(available, a)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no agents available (need at least 1 API key)")
	}

	// Stage 1
	fmt.Fprintln(os.Stderr, "Stage 1: Launching parallel agent analysis...")
	ctx1, cancel1 := context.WithTimeout(ctx, time.Duration(stage1Timeout)*time.Second)
	defer cancel1()

	fmt.Fprintf(os.Stderr, "  Waiting for agents (%ds timeout)...\n", stage1Timeout)
	start1 := time.Now()
	results := runStage1WithPrompt(ctx1, available, prompt)
	fmt.Fprintf(os.Stderr, "  Stage 1 duration: %.1fs\n", time.Since(start1).Seconds())

	// Tally results
	succeeded := 0
	for _, r := range results {
		if r.Err == nil {
			fmt.Fprintf(os.Stderr, "  %s: SUCCESS\n", r.Agent)
			succeeded++
		} else {
			fmt.Fprintf(os.Stderr, "  %s: FAILED (%v)\n", r.Agent, r.Err)
		}
	}
	fmt.Fprintf(os.Stderr, "  Agents completed: %d/%d succeeded\n", succeeded, len(available))
	if succeeded == 0 {
		return nil, fmt.Errorf("all agents failed (0/%d succeeded)", len(available))
	}

	// Stage 2
	fmt.Fprintln(os.Stderr, "\nStage 2: Chairman synthesis...")
	ctx2, cancel2 := context.WithTimeout(ctx, time.Duration(stage2Timeout)*time.Second)
	defer cancel2()

	chairmanPrompt := buildChairmanPrompt(prompt, results)
	start2 := time.Now()
	chairResult, err := RunStage2(ctx2, chairmen, chairmanPrompt)
	if err != nil {
		return nil, fmt.Errorf("stage 2 failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  %s: SUCCESS\n", chairResult.Agent)
	fmt.Fprintf(os.Stderr, "  Stage 2 duration: %.1fs\n", time.Since(start2).Seconds())

	return &ConsensusResult{
		Stage1Results:   results,
		ChairmanName:    chairResult.Agent,
		ChairmanOutput:  chairResult.Output,
		AgentsSucceeded: succeeded,
	}, nil
}

// RunConsensusWithBuilder is like RunConsensus but accepts a function to build
// the chairman prompt from stage 1 results (allowing mode-specific prompt building).
func RunConsensusWithBuilder(ctx context.Context, agents, chairmen []Agent, stage1Prompt string, buildChairman func([]AgentResult) string, stage1Timeout, stage2Timeout int) (*ConsensusResult, error) {
	// Filter available agents
	var available []Agent
	for _, a := range agents {
		if a.Available() {
			available = append(available, a)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no agents available (need at least 1 API key)")
	}

	// Stage 1
	fmt.Fprintln(os.Stderr, "Stage 1: Launching parallel agent analysis...")
	ctx1, cancel1 := context.WithTimeout(ctx, time.Duration(stage1Timeout)*time.Second)
	defer cancel1()

	fmt.Fprintf(os.Stderr, "  Waiting for agents (%ds timeout)...\n", stage1Timeout)
	start1 := time.Now()
	results := runStage1WithPrompt(ctx1, available, stage1Prompt)
	fmt.Fprintf(os.Stderr, "  Stage 1 duration: %.1fs\n", time.Since(start1).Seconds())

	// Tally results
	succeeded := 0
	for _, r := range results {
		if r.Err == nil {
			fmt.Fprintf(os.Stderr, "  %s: SUCCESS\n", r.Agent)
			succeeded++
		} else {
			fmt.Fprintf(os.Stderr, "  %s: FAILED (%v)\n", r.Agent, r.Err)
		}
	}
	fmt.Fprintf(os.Stderr, "  Agents completed: %d/%d succeeded\n", succeeded, len(available))
	if succeeded == 0 {
		return nil, fmt.Errorf("all agents failed (0/%d succeeded)", len(available))
	}

	// Stage 2
	fmt.Fprintln(os.Stderr, "\nStage 2: Chairman synthesis...")
	ctx2, cancel2 := context.WithTimeout(ctx, time.Duration(stage2Timeout)*time.Second)
	defer cancel2()

	chairmanPrompt := buildChairman(results)
	start2 := time.Now()
	chairResult, err := RunStage2(ctx2, chairmen, chairmanPrompt)
	if err != nil {
		return nil, fmt.Errorf("stage 2 failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  %s: SUCCESS\n", chairResult.Agent)
	fmt.Fprintf(os.Stderr, "  Stage 2 duration: %.1fs\n", time.Since(start2).Seconds())

	return &ConsensusResult{
		Stage1Results:   results,
		ChairmanName:    chairResult.Agent,
		ChairmanOutput:  chairResult.Output,
		AgentsSucceeded: succeeded,
	}, nil
}

func buildChairmanPrompt(originalPrompt string, results []AgentResult) string {
	succeeded := 0
	for _, r := range results {
		if r.Err == nil {
			succeeded++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Compile consensus from %d of %d analyses.\n\n", succeeded, len(results))
	for _, r := range results {
		if r.Err == nil {
			fmt.Fprintf(&b, "--- %s Analysis ---\n%s\n\n", r.Agent, r.Output)
		}
	}
	return b.String()
}
