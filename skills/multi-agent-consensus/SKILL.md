---
name: multi-agent-consensus
description: Reusable multi-agent consensus infrastructure using two-stage synthesis for diverse AI perspectives (Claude/Gemini/Codex)
---

# Multi-Agent Consensus

## Overview

Provides two-stage consensus synthesis from Claude, Gemini, and Codex:
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
consensus-synthesis.sh --mode=code-review \
  --base-sha="$BASE" --head-sha="$HEAD" \
  --plan-file="$PLAN" --description="$DESC"
```

**General prompt mode:**
```bash
consensus-synthesis.sh --mode=general-prompt \
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

**Stage 1 (30s timeout per agent):**
- Claude, Gemini, Codex analyze independently in parallel
- Each provides structured feedback
- Results collected from all successful agents

**Stage 2 (30s timeout):**
- Chairman (Claude → Gemini → Codex fallback) synthesizes consensus
- Groups issues by agreement
- Highlights disagreements explicitly
- Produces final three-tier report
