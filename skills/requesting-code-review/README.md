# Multi-Reviewer Code Review System

## Architecture

This directory contains a multi-reviewer code review system that coordinates parallel reviews from three AI reviewers:

1. **Claude Code** (via Task tool) - Required
2. **Gemini** (via CLI) - Optional
3. **Codex** (via MCP) - Optional

## How It Works

### Direct Invocation (Gemini only)

The `multi-review.sh` script can directly invoke:
- ✅ Gemini CLI (subprocess)
- ❌ Claude Task tool (requires Claude Code environment)
- ❌ Codex MCP (requires Claude Code environment)

### Claude Code Orchestration

When invoked from within a Claude Code session, the assistant:

1. Calls `multi-review.sh` to get Gemini's review
2. Simultaneously dispatches Claude subagent review (via Task tool)
3. Simultaneously calls Codex MCP (via mcp__codex-cli__codex tool)
4. Aggregates all three reviews into consensus report

## Files

- `multi-review.sh` - Main coordination script
- `SKILL.md` - Instructions for Claude Code assistant
- `code-reviewer.md` - Agent definition for Claude review
- `test-multi-review.sh` - Test suite
- `README.md` - This file

## Testing

Run the test suite:
```bash
./skills/requesting-code-review/test-multi-review.sh
```

Test with real code:
```bash
BASE_SHA=$(git rev-parse HEAD~1)
HEAD_SHA=$(git rev-parse HEAD)
./skills/requesting-code-review/multi-review.sh "$BASE_SHA" "$HEAD_SHA" "-" "Test review"
```

## Dependencies

- bash 4.0+
- git
- jq (for JSON parsing)
- bc (for calculations)
- gemini CLI (optional, for Gemini reviews)
- Claude Code (optional, for Claude/Codex reviews)
