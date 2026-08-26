package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func haikuServer(t *testing.T, responseText string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", 404)
			return
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing or wrong API key")
			http.Error(w, "unauthorized", 401)
			return
		}

		// Verify request body
		var body struct {
			Model    string `json:"model"`
			MaxTokens int   `json:"max_tokens"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}
		if body.Model != DefaultRouterModel {
			t.Errorf("expected model %s, got %s", DefaultRouterModel, body.Model)
		}

		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": responseText},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRouteHard(t *testing.T) {
	srv := haikuServer(t, "HARD")
	defer srv.Close()

	r := &Router{APIKey: "test-key", BaseURL: srv.URL}
	result, err := r.Route(context.Background(), "Build a reactive spreadsheet", BiasBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != "HARD" {
		t.Errorf("expected HARD, got %s", result.Classification)
	}
	if result.Model != DefaultHardModel {
		t.Errorf("expected %s, got %s", DefaultHardModel, result.Model)
	}
}

func TestRouteEasy(t *testing.T) {
	srv := haikuServer(t, "EASY")
	defer srv.Close()

	r := &Router{APIKey: "test-key", BaseURL: srv.URL}
	result, err := r.Route(context.Background(), "Add a REST endpoint", BiasBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != "EASY" {
		t.Errorf("expected EASY, got %s", result.Classification)
	}
	if result.Model != DefaultEasyModel {
		t.Errorf("expected %s, got %s", DefaultEasyModel, result.Model)
	}
}

func TestRouteOff(t *testing.T) {
	r := &Router{APIKey: "test-key"}
	result, err := r.Route(context.Background(), "anything", BiasOff)
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "" {
		t.Errorf("expected empty model for off, got %s", result.Model)
	}
}

func TestRouteEmptyBias(t *testing.T) {
	r := &Router{APIKey: "test-key"}
	result, err := r.Route(context.Background(), "anything", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "" {
		t.Errorf("expected empty model for empty bias, got %s", result.Model)
	}
}

func TestRouteInvalidBias(t *testing.T) {
	r := &Router{APIKey: "test-key"}
	_, err := r.Route(context.Background(), "anything", "turbo")
	if err == nil {
		t.Error("expected error for invalid bias")
	}
}

func TestRouteAPIFailureDefaultsToSonnet(t *testing.T) {
	// Server that returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()

	r := &Router{APIKey: "test-key", BaseURL: srv.URL}
	result, err := r.Route(context.Background(), "something", BiasBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != DefaultEasyModel {
		t.Errorf("expected fallback to %s, got %s", DefaultEasyModel, result.Model)
	}
}

func TestRouteNoAPIKey(t *testing.T) {
	r := &Router{APIKey: "", BaseURL: "http://localhost:1"}
	result, err := r.Route(context.Background(), "something", BiasBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Should gracefully fall back to Sonnet
	if result.Model != DefaultEasyModel {
		t.Errorf("expected fallback to %s, got %s", DefaultEasyModel, result.Model)
	}
}

func TestRouteCaseInsensitive(t *testing.T) {
	srv := haikuServer(t, "hard")
	defer srv.Close()

	r := &Router{APIKey: "test-key", BaseURL: srv.URL}
	result, err := r.Route(context.Background(), "task", BiasQuality)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != "HARD" {
		t.Errorf("expected HARD, got %s", result.Classification)
	}
}

func TestRouteWithExtraText(t *testing.T) {
	srv := haikuServer(t, "I think this is HARD because...")
	defer srv.Close()

	r := &Router{APIKey: "test-key", BaseURL: srv.URL}
	result, err := r.Route(context.Background(), "task", BiasBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != "HARD" {
		t.Errorf("expected HARD, got %s", result.Classification)
	}
}

func TestPromptForBias(t *testing.T) {
	tests := []struct {
		bias    string
		wantLen bool
	}{
		{BiasQuality, true},
		{BiasBalanced, true},
		{BiasCost, true},
		{"invalid", false},
	}
	for _, tt := range tests {
		p := PromptForBias(tt.bias)
		if tt.wantLen && p == "" {
			t.Errorf("expected non-empty prompt for %s", tt.bias)
		}
		if !tt.wantLen && p != "" {
			t.Errorf("expected empty prompt for %s", tt.bias)
		}
	}
}

func TestValidBias(t *testing.T) {
	valid := []string{"quality", "balanced", "cost", "off", ""}
	for _, b := range valid {
		if !ValidBias(b) {
			t.Errorf("expected %q to be valid", b)
		}
	}
	invalid := []string{"turbo", "fast", "QUALITY"}
	for _, b := range invalid {
		if ValidBias(b) {
			t.Errorf("expected %q to be invalid", b)
		}
	}
}

func TestAllBiasPromptsContainClassificationInstruction(t *testing.T) {
	for _, bias := range []string{BiasQuality, BiasBalanced, BiasCost} {
		p := PromptForBias(bias)
		if p == "" {
			t.Fatalf("empty prompt for %s", bias)
		}
		// All prompts should ask for HARD or EASY
		if !(contains(p, "HARD") && contains(p, "EASY")) {
			t.Errorf("prompt for %s missing HARD/EASY classification", bias)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestDefaultModelIDs pins the routing defaults so a model bump is a deliberate,
// reviewed change rather than a drift.
func TestDefaultModelIDs(t *testing.T) {
	if DefaultRouterModel != "claude-haiku-4-5" {
		t.Errorf("DefaultRouterModel = %q", DefaultRouterModel)
	}
	if DefaultHardModel != "claude-opus-5" {
		t.Errorf("DefaultHardModel = %q", DefaultHardModel)
	}
	if DefaultEasyModel != "claude-sonnet-5" {
		t.Errorf("DefaultEasyModel = %q", DefaultEasyModel)
	}
}
