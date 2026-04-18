package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/signalnine/conclave/internal/config"
	"github.com/signalnine/conclave/internal/routing"
	"github.com/spf13/cobra"
)

var routeCmd = &cobra.Command{
	Use:   "route [task description or file]",
	Short: "Classify task complexity and recommend a model",
	Long: `Uses Haiku to classify a task as HARD or EASY and recommends
an appropriate model (Opus for HARD, Sonnet for EASY).

The routing bias controls the cost/quality tradeoff:
  quality   - routes most tasks to Opus (high cost, high quality)
  balanced  - moderate routing (default)
  cost      - minimal Opus routing (low cost)
  off       - disable routing (print "OFF" and exit)

Examples:
  conclave route "Build a reactive spreadsheet with cycle detection"
  conclave route task-spec.md
  conclave route --bias=cost "Add a REST endpoint"`,
	Args: cobra.ExactArgs(1),
	RunE: runRoute,
}

func init() {
	routeCmd.Flags().String("bias", "", "Routing bias: quality, balanced, cost, off (overrides CONCLAVE_ROUTING)")
	rootCmd.AddCommand(routeCmd)
}

func runRoute(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	biasFlag, _ := cmd.Flags().GetString("bias")
	bias := biasFlag
	if bias == "" {
		bias = cfg.RoutingBias
	}
	if bias == "" {
		bias = routing.BiasBalanced
	}
	if bias == routing.BiasOff {
		fmt.Println("OFF (routing disabled)")
		return nil
	}
	if !routing.ValidBias(bias) {
		return fmt.Errorf("invalid routing bias: %q (valid: quality, balanced, cost, off)", bias)
	}

	// Read task: treat arg as file path first, then as inline text
	taskContent := args[0]
	if data, err := os.ReadFile(args[0]); err == nil {
		taskContent = string(data)
	}

	router := &routing.Router{
		APIKey:  cfg.AnthropicAPIKey,
		BaseURL: cfg.AnthropicBaseURL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := router.Route(ctx, taskContent, bias)
	if err != nil {
		return fmt.Errorf("routing failed: %w", err)
	}

	fmt.Printf("%s %s\n", result.Classification, result.Model)
	return nil
}
