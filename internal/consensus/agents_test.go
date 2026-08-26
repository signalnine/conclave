package consensus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/signalnine/conclave/internal/config"
)

func TestClaudeAgent_Available(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"with key", "sk-test", true},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClaudeAgent(&config.Config{AnthropicAPIKey: tt.key})
			if got := a.Available(); got != tt.want {
				t.Errorf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeAgent_Run(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-test" {
			t.Error("missing api key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing version header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "claude response"},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		AnthropicAPIKey:    "sk-test",
		AnthropicModel:     "claude-test",
		AnthropicMaxTokens: 100,
		AnthropicBaseURL:   srv.URL,
	}
	a := NewClaudeAgent(cfg)
	got, err := a.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "claude response" {
		t.Errorf("got %q", got)
	}
}

func TestClaudeAgent_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "rate limited"},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		AnthropicAPIKey:  "sk-test",
		AnthropicBaseURL: srv.URL,
	}
	_, err := NewClaudeAgent(cfg).Run(context.Background(), "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGLMAgent_Run(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer zh-test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "glm-5.3-flash" {
			t.Errorf("model = %v", body["model"])
		}
		if body["max_tokens"] != float64(4096) {
			t.Errorf("max_tokens = %v", body["max_tokens"])
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"reasoning_content": "thinking...",
					"content":           "glm response",
				}},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		GLMAPIKey:    "zh-test",
		GLMModel:     "glm-5.3-flash",
		GLMMaxTokens: 4096,
		GLMBaseURL:   srv.URL,
	}
	got, err := NewGLMAgent(cfg).Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "glm response" {
		t.Errorf("got %q", got)
	}
}

func TestGLMAgent_Run_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "1113", "message": "Insufficient balance or no resource package. Please recharge."},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{GLMAPIKey: "zh-test", GLMModel: "glm-5.3-flash", GLMMaxTokens: 100, GLMBaseURL: srv.URL}
	_, err := NewGLMAgent(cfg).Run(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "Insufficient balance") {
		t.Errorf("expected API error surfaced, got %v", err)
	}
}

func TestGLMAgent_Run_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"reasoning_content": "only thought", "content": ""}}},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{GLMAPIKey: "zh-test", GLMModel: "glm-5.3-flash", GLMMaxTokens: 100, GLMBaseURL: srv.URL}
	_, err := NewGLMAgent(cfg).Run(context.Background(), "test")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestGLMAgent_Available(t *testing.T) {
	if NewGLMAgent(&config.Config{GLMAPIKey: ""}).Available() {
		t.Error("should not be available without key")
	}
	if !NewGLMAgent(&config.Config{GLMAPIKey: "key"}).Available() {
		t.Error("should be available with key")
	}
}

func TestCodexAgent_Run_ResponsesAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer op-test" {
			t.Error("missing auth header")
		}
		if r.URL.Path != "/v1/responses" {
			t.Errorf("path = %q, want /v1/responses", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{"type": "message", "content": []map[string]any{
					{"type": "text", "text": "codex response"},
				}},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		OpenAIAPIKey:    "op-test",
		OpenAIModel:     "gpt-5.1-codex-max",
		OpenAIMaxTokens: 100,
		OpenAIBaseURL:   srv.URL,
	}
	got, err := NewCodexAgent(cfg).Run(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if got != "codex response" {
		t.Errorf("got %q", got)
	}
}

func TestCodexAgent_Run_ChatCompletions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "chat response"}},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		OpenAIAPIKey:  "op-test",
		OpenAIModel:   "gpt-4o",
		OpenAIBaseURL: srv.URL,
	}
	got, err := NewCodexAgent(cfg).Run(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if got != "chat response" {
		t.Errorf("got %q", got)
	}
}

func TestAgent_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	cfg := &config.Config{
		AnthropicAPIKey:  "sk-test",
		AnthropicBaseURL: srv.URL,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewClaudeAgent(cfg).Run(ctx, "test")
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
