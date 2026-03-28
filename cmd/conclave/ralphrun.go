package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signalnine/conclave/internal/bus"
	gitpkg "github.com/signalnine/conclave/internal/git"
	"github.com/signalnine/conclave/internal/ralph"
	"github.com/spf13/cobra"
)

var ralphRunCmd = &cobra.Command{
	Use:   "ralph-run",
	Short: "Autonomous retry loop for tasks",
	Long:  "Runs a task through implement/test/spec gates with automatic retries and stuck detection.",
	RunE:  runRalphRun,
}

func init() {
	ralphRunCmd.Flags().String("task", "", "Task description or prompt file (required)")
	ralphRunCmd.Flags().Int("max-iterations", 5, "Maximum retry iterations")
	ralphRunCmd.Flags().Int("implement-timeout", 300, "Implementation gate timeout (seconds)")
	ralphRunCmd.Flags().Int("test-timeout", 120, "Test gate timeout (seconds)")
	ralphRunCmd.Flags().Int("spec-timeout", 120, "Spec gate timeout (seconds)")
	ralphRunCmd.Flags().Int("stuck-threshold", 3, "Consecutive same-error count before strategy shift")
	ralphRunCmd.Flags().Bool("skip-spec", false, "Skip spec compliance gate")
	ralphRunCmd.Flags().String("board-dir", "", "Bulletin board directory for cross-task communication")
	ralphRunCmd.Flags().String("board-topic", "", "Topic to publish board messages to")
	ralphRunCmd.Flags().String("task-id", "", "Task identifier for board messages")
	ralphRunCmd.Flags().Bool("eval", false, "Enable evaluator gate after test failure")
	ralphRunCmd.Flags().String("eval-model", "", "Model for evaluator (default: same as generator)")
	ralphRunCmd.Flags().Int("eval-timeout", 120, "Evaluator gate timeout (seconds)")
	ralphRunCmd.Flags().String("system-prompt", "", "Custom system prompt (prepended to TDDPreamble)")
	rootCmd.AddCommand(ralphRunCmd)
}

func runRalphRun(cmd *cobra.Command, args []string) error {
	task, _ := cmd.Flags().GetString("task")
	maxIter, _ := cmd.Flags().GetInt("max-iterations")
	implTimeout, _ := cmd.Flags().GetInt("implement-timeout")
	testTimeout, _ := cmd.Flags().GetInt("test-timeout")
	stuckThreshold, _ := cmd.Flags().GetInt("stuck-threshold")
	skipSpec, _ := cmd.Flags().GetBool("skip-spec")
	boardDir, _ := cmd.Flags().GetString("board-dir")
	boardTopic, _ := cmd.Flags().GetString("board-topic")
	taskID, _ := cmd.Flags().GetString("task-id")
	evalEnabled, _ := cmd.Flags().GetBool("eval")
	evalModel, _ := cmd.Flags().GetString("eval-model")
	evalTimeout, _ := cmd.Flags().GetInt("eval-timeout")
	systemPrompt, _ := cmd.Flags().GetString("system-prompt")

	if task == "" {
		return fmt.Errorf("--task is required")
	}

	cwd, _ := os.Getwd()
	lock := ralph.NewLock(cwd)
	if err := lock.Acquire(); err != nil {
		return err
	}
	defer lock.Release()

	sm := ralph.NewStateManager(cwd)
	stateTaskID := fmt.Sprintf("ralph-%d", time.Now().Unix())
	if err := sm.Init(stateTaskID, maxIter); err != nil {
		return err
	}
	defer sm.Cleanup()

	g := gitpkg.New(cwd)
	ctx := context.Background()

	for {
		state, err := sm.Load()
		if err != nil {
			return err
		}

		if state.Iteration > state.MaxIterations {
			fmt.Fprintf(os.Stderr, "\nMax iterations (%d) reached. Branching failed work.\n", maxIter)
			ralph.BranchFailedWork(g, stateTaskID, state)
			return fmt.Errorf("max iterations reached")
		}

		fmt.Fprintf(os.Stderr, "\n=== Ralph Loop: Iteration %d/%d ===\n", state.Iteration, state.MaxIterations)

		// Check if stuck
		stuckDirective := ""
		if ralph.IsStuck(state.StuckCount, stuckThreshold) {
			fmt.Fprintln(os.Stderr, "STUCK DETECTED - forcing strategy shift")
			sm.IncrementStrategyShift()
			stuckDirective = ralph.StuckDirective
		}

		// Gate 1: Implementation
		fmt.Fprintln(os.Stderr, "Gate 1: Implementation...")
		preamble := ralph.TDDPreamble
		if systemPrompt != "" {
			preamble = systemPrompt + "\n\n" + ralph.TDDPreamble
		}
		prompt := preamble + "\n\n" + task
		if stuckDirective != "" {
			prompt = stuckDirective + "\n\n" + prompt
		}
		ctxContent, _ := os.ReadFile(sm.ContextFile())
		if len(ctxContent) > 0 {
			prompt = prompt + "\n\n## Previous Attempt Context\n" + string(ctxContent)
		}

		// Read board at iteration start
		if boardDir != "" {
			entries, err := ralph.ReadBoard(boardDir, 20)
			if err == nil && len(entries) > 0 {
				boardCtx := ralph.FormatBoardContext(entries)
				prompt = prompt + "\n\n" + boardCtx
			}
		}

		implCtx, implCancel := context.WithTimeout(ctx, time.Duration(implTimeout)*time.Second)
		implCmd := exec.CommandContext(implCtx, "claude", "-p", prompt)
		implCmd.Dir = cwd
		implOut, implErr := implCmd.CombinedOutput()
		implCancel()

		iterationOutput := string(implOut)

		// Write board markers from iteration output
		if boardDir != "" && boardTopic != "" {
			markers := ralph.ExtractBusMarkers(iterationOutput)
			if len(markers) > 0 {
				fileBus, busErr := bus.NewFileBus(boardDir, 100*time.Millisecond, time.Second)
				if busErr == nil {
					senderID := taskID
					if senderID == "" {
						senderID = "ralph"
					}
					ralph.PublishMarkers(fileBus, boardTopic, senderID, markers)
					fileBus.Close()
				}
			}
		}

		if implErr != nil {
			fmt.Fprintf(os.Stderr, "  Implementation exited with error: %v\n", implErr)
			// Check if code was written despite non-zero exit (rate limits, session limits, etc.)
			status, statusErr := g.StatusPorcelain()
			if statusErr != nil || strings.TrimSpace(status) == "" {
				fmt.Fprintln(os.Stderr, "  No file changes detected, skipping to next iteration")
				sm.Update("implement", 1, iterationOutput)
				continue
			}
			fmt.Fprintln(os.Stderr, "  File changes detected despite error, proceeding to test gate")
		} else {
			fmt.Fprintln(os.Stderr, "  Implementation complete")
		}

		// Gate 2: Tests
		fmt.Fprintln(os.Stderr, "Gate 2: Tests...")
		testOutput, testErr := ralph.RunTestGate(ctx, cwd, testTimeout)
		if testErr != nil {
			fmt.Fprintf(os.Stderr, "  Tests failed\n")
			if evalEnabled {
				fmt.Fprintln(os.Stderr, "  Running evaluator...")
				evalOutput, evalErr := ralph.RunEvalGate(ctx, cwd, task, testOutput, evalModel, evalTimeout)
				if evalErr != nil {
					fmt.Fprintf(os.Stderr, "  Evaluator failed, using raw test output: %v\n", evalErr)
					sm.Update("tests", 1, testOutput)
				} else {
					fmt.Fprintln(os.Stderr, "  Evaluator feedback received")
					// Save raw test output to sidecar file
					rawRef := fmt.Sprintf(".ralph_raw_%d.txt", state.Iteration)
					if writeErr := os.WriteFile(filepath.Join(cwd, rawRef), []byte(testOutput), 0644); writeErr != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to save raw output: %v\n", writeErr)
				}
					// Hash raw test output for stuck detection
					rawLines := strings.Split(testOutput, "\n")
					if len(rawLines) > 20 {
						rawLines = rawLines[:20]
					}
					rawHash := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(rawLines, "\n"))))
					sm.UpdateWithEval("tests", 1, evalOutput, true, rawRef, rawHash)
				}
			} else {
				sm.Update("tests", 1, testOutput)
			}
			continue
		}
		fmt.Fprintln(os.Stderr, "  Tests passed")

		// Gate 3: Spec (optional)
		if !skipSpec {
			fmt.Fprintln(os.Stderr, "Gate 3: Spec compliance...")
			if strings.Contains(testOutput, "SPEC_PASS") || strings.Contains(iterationOutput, "SPEC_PASS") {
				fmt.Fprintln(os.Stderr, "  Spec compliance confirmed")
			}
		}

		// All gates passed — commit the work
		fmt.Fprintln(os.Stderr, "\nAll gates passed! Committing...")
		if err := g.AddAll(); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: git add failed: %v\n", err)
		}
		if g.HasStagedChanges() {
			if err := g.Commit(fmt.Sprintf("feat: %s (ralph-loop iteration %d)", stateTaskID, state.Iteration)); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: git commit failed: %v\n", err)
			}
		}
		fmt.Fprintln(os.Stderr, "Task complete.")
		return nil
	}
}
