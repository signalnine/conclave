---
name: multi-agent-consensus
description: Use when you need diverse AI perspectives via two-stage synthesis (Claude/GLM/Codex)
---

# Multi-Agent Consensus

## Overview

Provides two-stage consensus synthesis from Claude, GLM, and Codex:
1. **Stage 1:** Independent parallel analysis from each agent
2. **Stage 2:** Chairman agent synthesizes final consensus

Groups responses by agreement level and explicitly highlights disagreements.

## When to Use

Use when you need diverse AI perspectives to reduce bias and blind spots:
- Design validation (brainstorming)
- Code review (requesting-code-review)
- Root cause analysis (debugging)
- Verification checks (before completion)

## Interface

**Code review mode:**
```bash
conclave consensus --mode=code-review \
  --base-sha="$BASE" --head-sha="$HEAD" \
  --plan-file="$PLAN" --description="$DESC"
```

**General prompt mode:**
```bash
conclave consensus --mode=general-prompt \
  --prompt="Your question here" \
  --context="Optional background info"
```

## Output

Three-tier consensus report:
- **High Priority** - Multiple reviewers agree
- **Medium Priority** - Single reviewer, significant issue
- **Consider** - Suggestions from any reviewer

Consensus saved to `/tmp/consensus-XXXXXX.md` with full context and all Stage 1 analyses.

## How It Works

**Stage 1 (60s timeout per agent, configurable):**
- Claude, GLM, Codex analyze independently in parallel
- Each provides structured feedback
- Results collected from all successful agents

**Stage 2 (60s timeout, configurable):**
- Chairman (Claude → GLM → Codex fallback) synthesizes consensus
- Groups issues by agreement
- Highlights disagreements explicitly
- Produces final three-tier report

## Configuration

**Timeout Configuration:**
```bash
# Via environment variables
export CONSENSUS_STAGE1_TIMEOUT=90
export CONSENSUS_STAGE2_TIMEOUT=90

# Via CLI flags
conclave consensus --mode=general-prompt \
  --prompt="Your question" \
  --stage1-timeout=90 \
  --stage2-timeout=90
```

**Default timeouts:** 60 seconds per stage
- Covers P95-P99 API latency scenarios
- Adjust higher for very complex prompts or slow networks
- Adjust lower for simple prompts when speed is critical

## When It Fails

Stderr reports each agent (`Claude: SUCCESS`, `GLM: FAILED (API error: ...)`) and then `Agents completed: n/3 succeeded`. One failed agent does not stop the run.

| Message | Cause | What to do |
|---------|-------|------------|
| `no agents available (need at least 1 API key)` | None of `ANTHROPIC_API_KEY`, `ZHIPU_API_KEY` (or `ZAI_API_KEY`), `OPENAI_API_KEY` is set | Export at least one, or put it in `./.env` or `~/.env` (both load automatically) |
| `<Agent>: FAILED (API error: ...)`, run continues | Bad key, quota, or model error for that provider | Nothing required. Consensus proceeds with the agents that succeeded; fix the key to get that voice back |
| `<Agent>: FAILED (... context deadline exceeded)` | Stage 1 exceeded its timeout (default 60s) | Raise `--stage1-timeout` or `CONSENSUS_STAGE1_TIMEOUT`, or shorten the prompt and context |
| `all agents failed (0/n succeeded)` | Every provider errored or timed out | Check network and keys, rerun with a longer timeout. If it persists, do single-agent analysis and say consensus was unavailable |
| `stage 2 failed: all chairman agents failed` | Every chairman candidate (Claude, then GLM, then Codex) failed to synthesize | Rerun with `--stage2-timeout` raised. Stage 1 output is not saved on this path, so if it repeats, run the agents individually |
| `git diff: ...` (code-review mode) | Unknown SHA, or not run from inside the repository | Check both SHAs with `git rev-parse <sha>` from the repo root |
| `reading plan file "...": ...` | `--plan-file` path does not exist | Fix the path or omit the flag; it is optional |
| `--mode is required`, `... requires --base-sha, --head-sha, --description`, `... requires --prompt` | Missing flags | See Interface above. `--dry-run` validates arguments without calling any API |
