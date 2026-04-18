package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalnine/conclave/internal/bus"
	gitpkg "github.com/signalnine/conclave/internal/git"
	"github.com/signalnine/conclave/internal/parallel"
	"github.com/signalnine/conclave/internal/plan"
	"github.com/signalnine/conclave/internal/ralph"
	"github.com/spf13/cobra"
)

var parallelRunCmd = &cobra.Command{
	Use:   "parallel-run",
	Short: "Execute plan tasks in parallel using git worktrees",
	Long:  "Parses an implementation plan, computes wave dependencies, and executes tasks in parallel using git worktrees.",
	RunE:  runParallelRun,
}

func init() {
	parallelRunCmd.Flags().String("plan", "", "Path to implementation plan file (required)")
	parallelRunCmd.Flags().Int("max-concurrent", 3, "Maximum concurrent tasks")
	parallelRunCmd.Flags().Bool("dry-run", false, "Parse and validate plan only")
	rootCmd.AddCommand(parallelRunCmd)
}

func runParallelRun(cmd *cobra.Command, args []string) error {
	planFile, _ := cmd.Flags().GetString("plan")
	maxConc, _ := cmd.Flags().GetInt("max-concurrent")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if planFile == "" {
		return fmt.Errorf("--plan is required")
	}

	f, err := os.Open(planFile)
	if err != nil {
		return fmt.Errorf("opening plan: %w", err)
	}
	defer f.Close()

	tasks, err := plan.ParsePlan(f)
	if err != nil {
		return fmt.Errorf("parsing plan: %w", err)
	}

	if err := plan.Validate(tasks); err != nil {
		return fmt.Errorf("validating plan: %w", err)
	}

	tasks = plan.DetectFileOverlaps(tasks)
	if err := plan.Validate(tasks); err != nil {
		return fmt.Errorf("validating plan after file-overlap dep injection: %w", err)
	}
	waves := plan.ComputeWaves(tasks)
	waveCount := plan.WaveCount(waves)

	fmt.Fprintf(os.Stderr, "Plan: %d tasks, %d waves, max %d concurrent\n", len(tasks), waveCount, maxConc)

	if dryRun {
		fmt.Println("Dry run: Plan parsed and validated successfully")
		for w := 0; w < waveCount; w++ {
			waveTasks := plan.TasksInWave(tasks, waves, w)
			var names []string
			for _, t := range waveTasks {
				names = append(names, fmt.Sprintf("Task %d: %s", t.ID, t.Title))
			}
			fmt.Printf("  Wave %d: %s\n", w, strings.Join(names, ", "))
		}
		return nil
	}

	// Create bus directory for cross-task communication
	baseDir, _ := os.Getwd()
	busDir := filepath.Join(baseDir, ".conclave", "bus")
	if err := os.MkdirAll(busDir, 0755); err != nil {
		return fmt.Errorf("creating bus directory: %w", err)
	}
	defer os.RemoveAll(busDir)

	// Write PID file for stale detection
	_ = os.WriteFile(filepath.Join(busDir, ".pid"), []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	g := gitpkg.New(".")
	sched := parallel.NewScheduler(tasks, waves, maxConc)

	for wave := 0; wave < waveCount; wave++ {
		fmt.Fprintf(os.Stderr, "\n=== Wave %d/%d ===\n", wave+1, waveCount)

		// Create wave-specific board directory
		waveBusDir := filepath.Join(busDir, fmt.Sprintf("wave-%d", wave))
		if err := os.MkdirAll(waveBusDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create wave bus dir: %v\n", err)
		}

		worktreeDir := filepath.Join(baseDir, ".worktrees")
		cmdBuilder := func(worktree, taskSpec string, taskID int, boardDir, boardTopic string) *exec.Cmd {
			return parallel.BuildRalphCommand(worktree, taskSpec, taskID, boardDir, boardTopic)
		}

		if err := parallel.ExecuteWave(context.Background(), g, sched, tasks, wave, worktreeDir, waveBusDir, cmdBuilder); err != nil {
			fmt.Fprintf(os.Stderr, "  Wave %d error: %v\n", wave, err)
		}

		// After wave completes, summarize board for next wave
		hasMoreWaves := wave+1 < waveCount
		if hasMoreWaves {
			entries, _ := ralph.ReadBoard(waveBusDir, 10)
			if len(entries) > 0 {
				nextWaveBusDir := filepath.Join(busDir, fmt.Sprintf("wave-%d", wave+1))
				if err := os.MkdirAll(nextWaveBusDir, 0755); err == nil {
					fileBus, busErr := bus.NewFileBus(nextWaveBusDir, 100*time.Millisecond, time.Second)
					if busErr == nil {
						summary := ralph.FormatBoardContext(entries)
						payload, _ := json.Marshal(struct {
							Text string `json:"text"`
						}{Text: summary})
						nextTopic := fmt.Sprintf("parallel.wave-%d.board", wave+1)
						_ = fileBus.Publish(nextTopic, bus.Message{
							Type:    "board.context",
							Sender:  "orchestrator",
							Payload: json.RawMessage(payload),
						})
						fileBus.Close()
					}
				}
			}
		}
	}

	sched.PrintSummary()
	return nil
}
