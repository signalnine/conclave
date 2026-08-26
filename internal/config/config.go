package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	// API Keys
	AnthropicAPIKey string
	GLMAPIKey       string
	OpenAIAPIKey    string

	// Model config
	AnthropicModel     string
	AnthropicMaxTokens int
	GLMModel           string
	GLMMaxTokens       int
	OpenAIModel        string
	OpenAIMaxTokens    int

	// Timeouts (seconds)
	Stage1Timeout int
	Stage2Timeout int

	// OpenRouter (fallback)
	OpenRouterAPIKey      string
	OpenRouterBaseURL     string
	OpenRouterClaudeModel string
	OpenRouterGLMModel    string
	OpenRouterCodexModel  string

	// Base URLs (for testing - override API endpoints)
	AnthropicBaseURL string
	GLMBaseURL       string
	OpenAIBaseURL    string

	// Parallel runner
	MaxConcurrent     int
	WorktreeDir       string
	MaxConflictReruns int

	// Ralph loop
	RalphTimeoutImplement int
	RalphTimeoutTest      int
	RalphTimeoutSpec      int
	RalphTimeoutQuality   int
	RalphTimeoutGlobal    int
	RalphStuckThreshold   int

	// Model routing
	RoutingBias string // quality, balanced, cost, off
}

func Load() *Config {
	loadDotEnv()

	return &Config{
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		GLMAPIKey:       coalesce(os.Getenv("ZHIPU_API_KEY"), coalesce(os.Getenv("ZAI_API_KEY"), os.Getenv("GLM_API_KEY"))),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),

		AnthropicModel:     envOr("ANTHROPIC_MODEL", "claude-opus-5"),
		AnthropicMaxTokens: envInt("ANTHROPIC_MAX_TOKENS", 16000),
		GLMModel:           envOr("GLM_MODEL", "glm-5.3-flash"),
		GLMMaxTokens:       envInt("GLM_MAX_TOKENS", 16000),
		OpenAIModel:        envOr("OPENAI_MODEL", "gpt-5.6-sol"),
		OpenAIMaxTokens:    envInt("OPENAI_MAX_TOKENS", 16000),

		Stage1Timeout: envInt("CONSENSUS_STAGE1_TIMEOUT", 60),
		Stage2Timeout: envInt("CONSENSUS_STAGE2_TIMEOUT", 60),

		OpenRouterAPIKey:      os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterBaseURL:     envOr("OPENROUTER_BASE_URL", "https://openrouter.ai/api"),
		OpenRouterClaudeModel: envOr("OPENROUTER_CLAUDE_MODEL", "anthropic/claude-opus-5"),
		OpenRouterGLMModel:    envOr("OPENROUTER_GLM_MODEL", "z-ai/glm-5.3-flash"),
		OpenRouterCodexModel:  envOr("OPENROUTER_CODEX_MODEL", "openai/gpt-5.6-sol"),

		AnthropicBaseURL: envOr("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		GLMBaseURL:       envOr("GLM_BASE_URL", "https://api.z.ai/api/paas/v4"),
		OpenAIBaseURL:    envOr("OPENAI_BASE_URL", "https://api.openai.com"),

		MaxConcurrent:     envInt("PARALLEL_MAX_CONCURRENT", 3),
		WorktreeDir:       envOr("PARALLEL_WORKTREE_DIR", ".worktrees"),
		MaxConflictReruns: envInt("PARALLEL_MAX_CONFLICT_RERUNS", 2),

		RalphTimeoutImplement: envInt("RALPH_TIMEOUT_IMPLEMENT", 1200),
		RalphTimeoutTest:      envInt("RALPH_TIMEOUT_TEST", 600),
		RalphTimeoutSpec:      envInt("RALPH_TIMEOUT_SPEC", 300),
		RalphTimeoutQuality:   envInt("RALPH_TIMEOUT_QUALITY", 180),
		RalphTimeoutGlobal:    envInt("RALPH_TIMEOUT_GLOBAL", 3600),
		RalphStuckThreshold:   envInt("RALPH_STUCK_THRESHOLD", 3),

		RoutingBias: envOr("CONCLAVE_ROUTING", "balanced"),
	}
}

func loadDotEnv() {
	// Load ./.env first (local project overrides), then ~/.env (global defaults).
	// Since parseDotEnvFile only sets vars not already present, order determines priority.
	parseDotEnvFile(".env")

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	parseDotEnvFile(filepath.Join(home, ".env"))
}

func parseDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			k = strings.TrimSpace(k)
			k = strings.TrimPrefix(k, "export ")
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
