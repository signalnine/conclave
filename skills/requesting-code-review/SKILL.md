---
name: requesting-code-review
description: Use when completing tasks, implementing major features, or before merging to verify work meets requirements
---

# Requesting Code Review

Get a structured code review to catch issues before they cascade.

**Core principle:** Fresh eyes on code = maximum coverage.

## When to Request Review

**Mandatory:**
- After each task in subagent-driven development
- After completing major feature
- Before merge to main

**Optional but valuable:**
- When stuck (fresh perspectives)
- Before refactoring (baseline check)
- After fixing complex bug

## How to Request Review

### Default: Subagent Review

Dispatch a `conclave:code-reviewer` subagent via the Task tool to review your changes:

1. Commit your work: `git add -A && git commit -m 'implementation'`
2. Get the diff:
   ```bash
   BASE_SHA=$(git merge-base origin/main HEAD)
   git diff $BASE_SHA..HEAD
   ```
3. Dispatch the `conclave:code-reviewer` subagent via Task tool with the diff and description of what was implemented
4. Address HIGH and MEDIUM priority findings
5. Re-verify after fixes

### Optional: Multi-Agent Consensus Review

For higher-stakes reviews (before merge to main, critical systems), use multi-agent consensus. Requires API keys for Gemini and/or Codex in addition to Claude.

**1. Get git SHAs:**
```bash
BASE_SHA=$(git rev-parse HEAD~1)  # or origin/main
HEAD_SHA=$(git rev-parse HEAD)
```

**2. Run multi-agent consensus:**

```bash
conclave consensus --mode=code-review \
  --base-sha="$BASE_SHA" \
  --head-sha="$HEAD_SHA" \
  --plan-file="$PLAN_FILE" \
  --description="$DESCRIPTION"
```

The framework uses a two-stage process:
- **Stage 1:** Launches Claude, Gemini, and Codex reviewers in parallel for independent analysis
- **Stage 2:** Chairman agent synthesizes consensus
- Groups issues by agreement level:
  - **High Priority** - Multiple reviewers agree
  - **Medium Priority** - Single reviewer, significant issue
  - **Consider** - Suggestions from any reviewer

**3. Act on consensus feedback:**
- **All reviewers agree** → Fix immediately before proceeding
- **Majority flagged** → Fix unless you have strong reasoning otherwise
- **Single reviewer** → Consider, but use judgment
- Push back if feedback is wrong (with technical reasoning)

## Integration with Workflows

**Subagent-Driven Development:**
- Review after EACH task
- Fix issues before next task

**Executing Plans:**
- Review after each batch (3 tasks)
- Get feedback, apply, continue

**Ad-Hoc Development:**
- Review before merge
- Review when stuck

## Red Flags

**Never:**
- Skip review because "it's simple"
- Ignore Critical issues
- Proceed with unfixed high-priority issues
- Argue with valid technical feedback

**If reviewer is wrong:**
- Push back with technical reasoning
- Show code/tests that prove it works
- Request clarification

## Files

- `conclave consensus` - Multi-agent consensus framework (code-review mode)
- `code-reviewer.md` - Claude agent definition (for subagent review)
