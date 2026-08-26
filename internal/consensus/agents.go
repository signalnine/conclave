package consensus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/signalnine/conclave/internal/config"
)

// Agent runs a prompt against an LLM and returns the response text.
type Agent interface {
	Name() string
	Run(ctx context.Context, prompt string) (string, error)
	Available() bool
}

// --- Claude ---

type ClaudeAgent struct {
	cfg *config.Config
}

func NewClaudeAgent(cfg *config.Config) *ClaudeAgent {
	return &ClaudeAgent{cfg: cfg}
}

func (a *ClaudeAgent) Name() string    { return "Claude" }
func (a *ClaudeAgent) Available() bool { return a.cfg.AnthropicAPIKey != "" }

func (a *ClaudeAgent) Run(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":      a.cfg.AnthropicModel,
		"max_tokens": a.cfg.AnthropicMaxTokens,
		"messages":   []map[string]any{{"role": "user", "content": prompt}},
	}
	data, _ := json.Marshal(body)

	url := strings.TrimRight(a.cfg.AnthropicBaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", a.cfg.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct{ Text string }   `json:"content"`
		Error   *struct{ Message string } `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 || result.Content[0].Text == "" {
		return "", fmt.Errorf("empty response")
	}
	return result.Content[0].Text, nil
}

// --- GLM (Z.ai, OpenAI-compatible chat completions) ---

type GLMAgent struct {
	cfg *config.Config
}

func NewGLMAgent(cfg *config.Config) *GLMAgent {
	return &GLMAgent{cfg: cfg}
}

func (a *GLMAgent) Name() string    { return "GLM" }
func (a *GLMAgent) Available() bool { return a.cfg.GLMAPIKey != "" }

func (a *GLMAgent) Run(ctx context.Context, prompt string) (string, error) {
	// Thinking is always on for glm-5.3-flash and cannot be disabled;
	// max_tokens has to leave room for it. reasoning_content is ignored.
	body := map[string]any{
		"model":      a.cfg.GLMModel,
		"max_tokens": a.cfg.GLMMaxTokens,
		"messages":   []map[string]any{{"role": "user", "content": prompt}},
	}
	url := strings.TrimRight(a.cfg.GLMBaseURL, "/") + "/chat/completions"
	return postChatCompletions(ctx, url, a.cfg.GLMAPIKey, body)
}

// postChatCompletions sends an OpenAI-style chat completion request with a
// Bearer token and returns choices[0].message.content. Shared by every
// OpenAI-compatible endpoint (Z.ai, OpenRouter).
func postChatCompletions(ctx context.Context, url, apiKey string, body map[string]any) (string, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
		Error *struct{ Message string } `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response")
	}
	return result.Choices[0].Message.Content, nil
}

// --- Codex (OpenAI) ---

type CodexAgent struct {
	cfg *config.Config
}

func NewCodexAgent(cfg *config.Config) *CodexAgent {
	return &CodexAgent{cfg: cfg}
}

func (a *CodexAgent) Name() string    { return "Codex" }
func (a *CodexAgent) Available() bool { return a.cfg.OpenAIAPIKey != "" }

var codexModelRe = regexp.MustCompile(`^gpt-5.*-codex`)
var reasoningModelRe = regexp.MustCompile(`^(o1|o3|o4|gpt-5)`)
var chatModelRe = regexp.MustCompile(`^(gpt-[345]|o1|o3|o4)`)

func (a *CodexAgent) Run(ctx context.Context, prompt string) (string, error) {
	base := strings.TrimRight(a.cfg.OpenAIBaseURL, "/")
	var url string
	var body map[string]any

	if codexModelRe.MatchString(a.cfg.OpenAIModel) {
		// Responses API for Codex models
		url = base + "/v1/responses"
		body = map[string]any{
			"model": a.cfg.OpenAIModel,
			"input": []map[string]any{{"role": "user", "content": prompt}},
		}
	} else if chatModelRe.MatchString(a.cfg.OpenAIModel) {
		url = base + "/v1/chat/completions"
		tokenKey := "max_tokens"
		if reasoningModelRe.MatchString(a.cfg.OpenAIModel) {
			tokenKey = "max_completion_tokens"
		}
		body = map[string]any{
			"model":    a.cfg.OpenAIModel,
			tokenKey:   a.cfg.OpenAIMaxTokens,
			"messages": []map[string]any{{"role": "user", "content": prompt}},
		}
	} else {
		url = base + "/v1/completions"
		body = map[string]any{
			"model":      a.cfg.OpenAIModel,
			"max_tokens": a.cfg.OpenAIMaxTokens,
			"prompt":     prompt,
		}
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return a.extractResponse(respBody)
}

// --- OpenRouter (fallback for any model) ---

type OpenRouterAgent struct {
	cfg   *config.Config
	model string
	label string
}

func NewOpenRouterAgent(cfg *config.Config, model, label string) *OpenRouterAgent {
	return &OpenRouterAgent{cfg: cfg, model: model, label: label}
}

func (a *OpenRouterAgent) Name() string    { return a.label + " (OpenRouter)" }
func (a *OpenRouterAgent) Available() bool { return a.cfg.OpenRouterAPIKey != "" && a.model != "" }

func (a *OpenRouterAgent) Run(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":    a.model,
		"messages": []map[string]any{{"role": "user", "content": prompt}},
	}
	url := strings.TrimRight(a.cfg.OpenRouterBaseURL, "/") + "/v1/chat/completions"
	return postChatCompletions(ctx, url, a.cfg.OpenRouterAPIKey, body)
}

// --- FallbackAgent (tries primary, falls back to secondary on failure) ---

type FallbackAgent struct {
	primary  Agent
	fallback Agent
}

func NewFallbackAgent(primary, fallback Agent) *FallbackAgent {
	return &FallbackAgent{primary: primary, fallback: fallback}
}

func (f *FallbackAgent) Name() string { return f.primary.Name() }
func (f *FallbackAgent) Available() bool {
	return f.primary.Available() || f.fallback.Available()
}

func (f *FallbackAgent) Run(ctx context.Context, prompt string) (string, error) {
	if f.primary.Available() {
		out, err := f.primary.Run(ctx, prompt)
		if err == nil {
			return out, nil
		}
		fmt.Fprintf(os.Stderr, "    %s direct failed (%v), trying OpenRouter fallback...\n", f.primary.Name(), err)
	}
	if f.fallback.Available() {
		return f.fallback.Run(ctx, prompt)
	}
	return "", fmt.Errorf("both primary and fallback unavailable for %s", f.primary.Name())
}

func (a *CodexAgent) extractResponse(body []byte) (string, error) {
	// Try Responses API format
	if codexModelRe.MatchString(a.cfg.OpenAIModel) {
		var result struct {
			Output []struct {
				Type    string                  `json:"type"`
				Content []struct{ Text string } `json:"content"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			for _, o := range result.Output {
				if o.Type == "message" && len(o.Content) > 0 {
					return o.Content[0].Text, nil
				}
			}
		}
	}

	// Try chat completions format
	var chat struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chat); err == nil && len(chat.Choices) > 0 {
		if c := chat.Choices[0].Message.Content; c != "" {
			return c, nil
		}
	}

	// Try completions format
	var comp struct {
		Choices []struct{ Text string } `json:"choices"`
	}
	if err := json.Unmarshal(body, &comp); err == nil && len(comp.Choices) > 0 {
		if c := comp.Choices[0].Text; c != "" {
			return c, nil
		}
	}

	// Check for error
	var errResp struct {
		Error struct{ Message string } `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", errResp.Error.Message)
	}

	return "", fmt.Errorf("empty response")
}
