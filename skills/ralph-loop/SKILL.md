---
name: ralph-loop
description: Autonomous iteration wrapper - runs tasks until success or cap hit
---

# Ralph Loop

Autonomous iteration wrapper for subagent-driven-development.

## Overview

Runs each task in a loop until:
- Success: tests pass AND spec compliance verified
- Failure: iteration cap hit (default: 5)

## Usage

```bash
./skills/ralph-loop/ralph-runner.sh <task-id> <task-prompt-file> [max-iterations]
```

## See Also

- Design: `docs/plans/2026-01-21-ralph-loop-design.md`
