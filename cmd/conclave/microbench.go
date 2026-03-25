package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/signalnine/conclave/internal/microbench"
	"github.com/spf13/cobra"
)

var microbenchCmd = &cobra.Command{
	Use:   "microbench",
	Short: "Run behavioral micro-benchmarks against skill text",
	Long: `Run lightweight benchmarks that test whether skill text changes affect
agent behavioral compliance (TDD, verification, iteration patterns)
without running full Thunderdome benchmark suites.

Each benchmark is a directory containing:
  task.md                  — the task prompt
  setup.sh                 — workspace setup (optional)
  verify.sh                — completion check (optional)
  expected_behaviors.json  — behavioral expectations`,
	RunE: runMicrobench,
}

type benchRunResult struct {
	Label  string             `json:"label"`
	Result *microbench.Result `json:"result"`
}

func init() {
	microbenchCmd.Flags().String("benchmarks-dir", "./benchmarks", "path to benchmarks directory")
	microbenchCmd.Flags().String("skill-text", "", "path to SKILL.md file to prepend to prompts")
	microbenchCmd.Flags().String("compare", "", "path to second SKILL.md for A/B comparison")
	microbenchCmd.Flags().Int("timeout", 300, "per-benchmark timeout in seconds")
	microbenchCmd.Flags().Bool("json", false, "output structured JSON results")
	microbenchCmd.Flags().String("benchmark", "", "run only the named benchmark")
	rootCmd.AddCommand(microbenchCmd)
}

func runMicrobench(cmd *cobra.Command, args []string) error {
	benchDir, _ := cmd.Flags().GetString("benchmarks-dir")
	skillTextPath, _ := cmd.Flags().GetString("skill-text")
	comparePath, _ := cmd.Flags().GetString("compare")
	timeoutSecs, _ := cmd.Flags().GetInt("timeout")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	benchFilter, _ := cmd.Flags().GetString("benchmark")

	// Resolve benchmarks-dir to absolute path
	benchDir, err := filepath.Abs(benchDir)
	if err != nil {
		return fmt.Errorf("resolving benchmarks dir: %w", err)
	}

	// Discover benchmarks
	benchmarks, err := microbench.DiscoverBenchmarks(benchDir)
	if err != nil {
		return fmt.Errorf("discovering benchmarks: %w", err)
	}

	if len(benchmarks) == 0 {
		return fmt.Errorf("no benchmarks found in %s", benchDir)
	}

	// Filter if requested
	if benchFilter != "" {
		var filtered []microbench.Benchmark
		for _, b := range benchmarks {
			if b.Name == benchFilter {
				filtered = append(filtered, b)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("benchmark %q not found", benchFilter)
		}
		benchmarks = filtered
	}

	// Load skill text
	skillText := ""
	if skillTextPath != "" {
		data, err := os.ReadFile(skillTextPath)
		if err != nil {
			return fmt.Errorf("reading skill text: %w", err)
		}
		skillText = string(data)
	}

	// Load comparison skill text
	compareText := ""
	if comparePath != "" {
		data, err := os.ReadFile(comparePath)
		if err != nil {
			return fmt.Errorf("reading comparison skill text: %w", err)
		}
		compareText = string(data)
	}

	timeout := time.Duration(timeoutSecs) * time.Second
	ctx := context.Background()
	cmdBuilder := microbench.DefaultCmdBuilder()

	var allResults []benchRunResult

	fmt.Fprintf(os.Stderr, "Running %d benchmark(s)...\n\n", len(benchmarks))

	for _, bench := range benchmarks {
		// Run with primary skill text
		label := "primary"
		if skillTextPath != "" {
			label = filepath.Base(skillTextPath)
		}
		fmt.Fprintf(os.Stderr, "  %s [%s] ... ", bench.Name, label)

		result, err := microbench.RunBenchmark(ctx, bench, skillText, timeout, cmdBuilder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			continue
		}

		allResults = append(allResults, benchRunResult{Label: label, Result: result})

		if result.Passed {
			fmt.Fprintf(os.Stderr, "PASS (%.1fs)\n", result.DurationSeconds)
		} else {
			fmt.Fprintf(os.Stderr, "FAIL (%.1fs)\n", result.DurationSeconds)
			for _, v := range result.Violations {
				fmt.Fprintf(os.Stderr, "    - %s\n", v)
			}
		}

		// Run with comparison skill text if requested
		if comparePath != "" {
			compareLabel := filepath.Base(comparePath)
			fmt.Fprintf(os.Stderr, "  %s [%s] ... ", bench.Name, compareLabel)

			result2, err := microbench.RunBenchmark(ctx, bench, compareText, timeout, cmdBuilder)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				continue
			}

			allResults = append(allResults, benchRunResult{Label: compareLabel, Result: result2})

			if result2.Passed {
				fmt.Fprintf(os.Stderr, "PASS (%.1fs)\n", result2.DurationSeconds)
			} else {
				fmt.Fprintf(os.Stderr, "FAIL (%.1fs)\n", result2.DurationSeconds)
				for _, v := range result2.Violations {
					fmt.Fprintf(os.Stderr, "    - %s\n", v)
				}
			}
		}
	}

	// Output
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(allResults); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	} else {
		// Summary
		fmt.Fprintf(os.Stderr, "\n--- Summary ---\n")
		passed := 0
		failed := 0
		for _, r := range allResults {
			if r.Result.Passed {
				passed++
			} else {
				failed++
			}
		}
		fmt.Fprintf(os.Stderr, "Passed: %d  Failed: %d  Total: %d\n", passed, failed, passed+failed)

		if comparePath != "" {
			fmt.Fprintf(os.Stderr, "\n--- A/B Comparison ---\n")
			printComparison(os.Stderr, allResults)
		}
	}

	// Exit with error code if any failures
	for _, r := range allResults {
		if !r.Result.Passed {
			os.Exit(1)
		}
	}
	return nil
}

func printComparison(w *os.File, results []benchRunResult) {
	// Group by benchmark name
	type pair struct {
		a, b *microbench.Result
	}
	pairs := map[string]*pair{}

	for i, r := range results {
		name := r.Result.Benchmark
		if _, ok := pairs[name]; !ok {
			pairs[name] = &pair{}
		}
		if i%2 == 0 {
			pairs[name].a = r.Result
		} else {
			pairs[name].b = r.Result
		}
	}

	for name, p := range pairs {
		if p.a == nil || p.b == nil {
			continue
		}
		fmt.Fprintf(w, "\n  %s:\n", name)
		fmt.Fprintf(w, "    %-30s  A        B\n", "")
		fmt.Fprintf(w, "    %-30s  %-8v %-8v\n", "passed", p.a.Passed, p.b.Passed)
		fmt.Fprintf(w, "    %-30s  %-8v %-8v\n", "task_completed", p.a.TaskCompleted, p.b.TaskCompleted)
		fmt.Fprintf(w, "    %-30s  %-8v %-8v\n", "tdd_compliance", p.a.Behaviors.TDDCompliance, p.b.Behaviors.TDDCompliance)
		fmt.Fprintf(w, "    %-30s  %-8v %-8v\n", "verification_before_commit", p.a.Behaviors.VerificationBeforeCommit, p.b.Behaviors.VerificationBeforeCommit)
		fmt.Fprintf(w, "    %-30s  %-8d %-8d\n", "test_runs", p.a.Behaviors.TestRunCount, p.b.Behaviors.TestRunCount)
		fmt.Fprintf(w, "    %-30s  %-8d %-8d\n", "fix_cycles", p.a.Behaviors.FixCycles, p.b.Behaviors.FixCycles)
		fmt.Fprintf(w, "    %-30s  %-8.1f %-8.1f\n", "duration_s", p.a.DurationSeconds, p.b.DurationSeconds)
	}
}
