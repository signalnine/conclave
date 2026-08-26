package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Bias presets control the cost/quality tradeoff.
const (
	BiasQuality  = "quality"  // Routes most tasks to Opus (high cost, high quality)
	BiasBalanced = "balanced" // Moderate routing -- requires strong signals for Opus
	BiasCost     = "cost"     // Minimal Opus routing (low cost, Sonnet-default)
	BiasOff      = "off"      // No routing -- use caller's default model
)

// Default models.
const (
	DefaultRouterModel = "claude-haiku-4-5"
	DefaultHardModel   = "claude-opus-5"
	DefaultEasyModel   = "claude-sonnet-5"
)

// Router classifies task complexity and selects an appropriate model.
type Router struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Result holds the routing decision.
type Result struct {
	Model          string // Selected model ID
	Classification string // "HARD" or "EASY"
}

// Route classifies a task and returns the recommended model.
// Returns empty Result if bias is "off" or empty (caller should use its default).
func (r *Router) Route(ctx context.Context, taskDescription, bias string) (Result, error) {
	if bias == BiasOff || bias == "" {
		return Result{}, nil
	}

	prompt := PromptForBias(bias)
	if prompt == "" {
		return Result{}, fmt.Errorf("unknown routing bias: %q (valid: quality, balanced, cost, off)", bias)
	}

	fullPrompt := prompt + "\n\n=== TASK DESCRIPTION ===\n" + taskDescription

	response, err := r.callHaiku(ctx, fullPrompt)
	if err != nil {
		// Default to easy model on router failure -- don't block the task
		return Result{Model: DefaultEasyModel, Classification: "EASY"}, nil
	}

	if strings.Contains(strings.ToUpper(response), "HARD") {
		return Result{Model: DefaultHardModel, Classification: "HARD"}, nil
	}
	return Result{Model: DefaultEasyModel, Classification: "EASY"}, nil
}

func (r *Router) callHaiku(ctx context.Context, prompt string) (string, error) {
	if r.APIKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY required for routing")
	}

	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	body := map[string]interface{}{
		"model":      DefaultRouterModel,
		"max_tokens": 16,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", r.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("haiku API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from haiku")
	}

	return result.Content[0].Text, nil
}

// PromptForBias returns the classifier prompt for a given bias level.
func PromptForBias(bias string) string {
	switch bias {
	case BiasQuality:
		return promptQuality
	case BiasBalanced:
		return promptBalanced
	case BiasCost:
		return promptCost
	default:
		return ""
	}
}

// ValidBias returns true if the bias string is a recognized value.
func ValidBias(bias string) bool {
	switch bias {
	case BiasQuality, BiasBalanced, BiasCost, BiasOff, "":
		return true
	}
	return false
}

const promptQuality = `You are a task complexity classifier. Read the task description below and classify it as HARD or EASY.

A task is HARD if it has ANY of these characteristics:
- Complex state management with multiple interacting components that must stay consistent
- Concurrent or async operations with ordering constraints
- Algorithmic reasoning requiring careful logic (scheduling, constraint solving, graph traversal)
- Ambiguous specifications requiring significant inference about intended behavior
- Multiple subsystems that must coordinate (e.g., real-time updates + persistence + API)
- Complex data transformations with edge cases (overflow, empty, boundary conditions)

A task is EASY if it is:
- A straightforward CRUD feature or API endpoint
- A bug fix with clear reproduction steps
- A well-specified feature with clear inputs/outputs
- Adding tests, documentation, or configuration
- Simple refactoring or code cleanup

Respond with ONLY the single word HARD or EASY. Nothing else.`

const promptBalanced = `You are a task complexity classifier deciding whether to use an expensive AI model (HARD) or a cheap one (EASY). The expensive model costs 2x more. Only classify as HARD when the cheap model would genuinely struggle.

HARD means the task requires ALL THREE of:
1. Multiple interacting stateful components that must maintain consistency
2. Non-obvious algorithmic reasoning (not just "implement this spec")
3. Edge cases that require deep reasoning to discover (not just listed in the spec)

EASY means ANY of:
- The spec is clear and complete (even if the implementation is large)
- The task is primarily about building features to a well-defined interface
- Bug fixes, even complex ones with clear reproduction steps
- The complexity is in volume of code, not in reasoning about correctness

Most programming tasks are EASY. Reserve HARD for tasks where a junior developer would produce subtly incorrect code even with the full spec in front of them.

Respond with ONLY the single word HARD or EASY. Nothing else.`

const promptCost = `Classify this programming task as HARD or EASY to decide model routing.

HARD tasks (use expensive model ~5/19 tasks): Tasks where correctness requires reasoning about emergent behavior across interacting components. Examples:
- A reactive spreadsheet where cell updates must propagate without cycles
- A constraint scheduler where adding one constraint can invalidate others
- A circuit debugger requiring signal propagation analysis
- An analytics dashboard with real-time aggregations that must stay consistent

EASY tasks (use cheap model ~14/19 tasks): Tasks where the spec fully describes what to build, even if the implementation is substantial. Examples:
- Building a REST API with CRUD operations (even with auth, pagination, etc.)
- Implementing a search engine with well-defined indexing and query specs
- Bug fixes with clear reproduction steps
- Feature additions to existing codebases with clear interfaces
- A plugin marketplace with well-defined install/uninstall lifecycle

Default to EASY. Only say HARD if the task genuinely requires reasoning about subtle interactions between components.

Respond with ONLY the single word HARD or EASY. Nothing else.`
