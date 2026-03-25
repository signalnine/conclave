package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/signalnine/conclave/internal/analyze"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze behavioral signals from trial traces and correlate with scores",
	Long: `Analyze Thunderdome trial NDJSON traces to extract behavioral signals
(TDD compliance, verification patterns, iteration counts) and correlate
them with composite scores to identify which agent behaviors predict
high task performance.`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().String("results-dir", "", "path to Thunderdome results directory (contains runs/)")
	analyzeCmd.Flags().String("run", "", "analyze a specific run (timestamp)")
	analyzeCmd.Flags().String("orchestrator", "", "filter to one orchestrator")
	analyzeCmd.Flags().String("task", "", "filter to one task")
	analyzeCmd.Flags().Bool("correlate", false, "show correlation report (default mode)")
	analyzeCmd.Flags().Bool("csv", false, "output CSV format")
	analyzeCmd.Flags().Bool("json", false, "output per-trial JSON")
	analyzeCmd.Flags().Bool("by-task", false, "show per-task correlation breakdown")
	analyzeCmd.Flags().Bool("by-orchestrator", false, "show per-orchestrator correlation breakdown")
	analyzeCmd.Flags().Int("min-trials", 10, "minimum traced trials for correlation")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	resultsDir, _ := cmd.Flags().GetString("results-dir")
	if resultsDir == "" {
		return fmt.Errorf("--results-dir is required (path to Thunderdome results directory)")
	}

	runsDir := resultsDir + "/runs"
	flagRun, _ := cmd.Flags().GetString("run")
	flagOrchestrator, _ := cmd.Flags().GetString("orchestrator")
	flagTask, _ := cmd.Flags().GetString("task")
	flagCSV, _ := cmd.Flags().GetBool("csv")
	flagJSON, _ := cmd.Flags().GetBool("json")
	flagByTask, _ := cmd.Flags().GetBool("by-task")
	flagByOrchestrator, _ := cmd.Flags().GetBool("by-orchestrator")
	flagMinTrials, _ := cmd.Flags().GetInt("min-trials")

	// Validate runs dir exists
	if _, err := os.Stat(runsDir); err != nil {
		return fmt.Errorf("runs directory not found: %s", runsDir)
	}

	// If --run specified, validate it exists
	if flagRun != "" {
		runPath := runsDir + "/" + flagRun
		if _, err := os.Stat(runPath); err != nil {
			return fmt.Errorf("run directory not found: %s", flagRun)
		}
	}

	trials, err := analyze.DiscoverTrials(runsDir)
	if err != nil {
		return fmt.Errorf("discovering trials: %w", err)
	}

	// Apply filters
	if flagRun != "" {
		trials = filterByRun(trials, flagRun)
	}
	if flagOrchestrator != "" {
		trials = filterByOrchestrator(trials, flagOrchestrator)
	}
	if flagTask != "" {
		trials = filterByTask(trials, flagTask)
	}

	if len(trials) == 0 {
		return fmt.Errorf("no trials match the given filters")
	}

	// Output mode
	switch {
	case flagCSV:
		analyze.FormatCSV(os.Stdout, trials)
	case flagJSON:
		for i := range trials {
			analyze.FormatTrialJSON(os.Stdout, &trials[i])
		}
	case flagByTask:
		return formatByGroup(trials, flagMinTrials, groupByTask)
	case flagByOrchestrator:
		return formatByGroup(trials, flagMinTrials, groupByOrchestrator)
	default:
		// Correlation report (default)
		report := analyze.Correlate(trials)
		if report.TracedTrials < flagMinTrials {
			fmt.Fprintf(os.Stderr, "Warning: only %d traced trials (minimum %d recommended for reliable correlations)\n\n",
				report.TracedTrials, flagMinTrials)
		}
		analyze.FormatCorrelationTable(os.Stdout, report)
	}

	return nil
}

type groupFunc func(analyze.TrialAnalysis) string

func groupByTask(t analyze.TrialAnalysis) string         { return t.Task }
func groupByOrchestrator(t analyze.TrialAnalysis) string  { return t.Orchestrator }

func formatByGroup(trials []analyze.TrialAnalysis, minTrials int, fn groupFunc) error {
	groups := map[string][]analyze.TrialAnalysis{}
	for _, t := range trials {
		key := fn(t)
		groups[key] = append(groups[key], t)
	}

	for name, group := range groups {
		fmt.Fprintf(os.Stdout, "\n=== %s ===\n", name)
		report := analyze.Correlate(group)
		if report.TracedTrials < minTrials {
			fmt.Fprintf(os.Stderr, "  Warning: %s has only %d traced trials\n", name, report.TracedTrials)
		}
		analyze.FormatCorrelationTable(os.Stdout, report)
	}
	return nil
}

func filterByRun(trials []analyze.TrialAnalysis, run string) []analyze.TrialAnalysis {
	var out []analyze.TrialAnalysis
	for _, t := range trials {
		if t.RunTimestamp == run {
			out = append(out, t)
		}
	}
	return out
}

func filterByOrchestrator(trials []analyze.TrialAnalysis, orch string) []analyze.TrialAnalysis {
	var out []analyze.TrialAnalysis
	for _, t := range trials {
		if strings.Contains(t.Orchestrator, orch) {
			out = append(out, t)
		}
	}
	return out
}

func filterByTask(trials []analyze.TrialAnalysis, task string) []analyze.TrialAnalysis {
	var out []analyze.TrialAnalysis
	for _, t := range trials {
		if strings.Contains(t.Task, task) {
			out = append(out, t)
		}
	}
	return out
}
