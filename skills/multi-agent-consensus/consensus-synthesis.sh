#!/usr/bin/env bash
set -euo pipefail

# consensus-synthesis.sh
# Two-stage consensus synthesis for multi-agent analysis
# Stage 1: Parallel independent analysis from Claude, Gemini, Codex
# Stage 2: Chairman synthesizes consensus from all responses

#############################################
# Usage and Help
#############################################

usage() {
    cat <<EOF
Usage: consensus-synthesis.sh [OPTIONS]

Two-stage consensus synthesis for multi-agent code review and analysis.

MODES:
  --mode=code-review       Review code changes between two commits
  --mode=general-prompt    Analyze a general question with context

CODE REVIEW MODE OPTIONS:
  --base-sha=SHA          Base commit SHA (required)
  --head-sha=SHA          Head commit SHA (required)
  --description=TEXT      Change description (required)
  --plan-file=PATH        Optional: Path to implementation plan file

GENERAL PROMPT MODE OPTIONS:
  --prompt=TEXT           Question or prompt to analyze (required)
  --context=TEXT          Optional: Additional context for analysis

COMMON OPTIONS:
  --dry-run               Parse arguments and validate, but don't execute
  --help                  Show this help message

EXAMPLES:
  # Code review
  consensus-synthesis.sh --mode=code-review \\
    --base-sha=abc123 \\
    --head-sha=def456 \\
    --description="Add authentication" \\
    --plan-file=docs/plans/auth.md

  # General prompt
  consensus-synthesis.sh --mode=general-prompt \\
    --prompt="What could go wrong with this design?" \\
    --context="\$(cat design.md)"

OUTPUT:
  - Console: Progress updates and final consensus
  - File: Detailed breakdown saved to /tmp/consensus-XXXXXX.md

EOF
}

#############################################
# Argument Parsing
#############################################

# Initialize variables
MODE=""
BASE_SHA=""
HEAD_SHA=""
DESCRIPTION=""
PLAN_FILE=""
PROMPT=""
CONTEXT=""
DRY_RUN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode=*)
            MODE="${1#*=}"
            shift
            ;;
        --base-sha=*)
            BASE_SHA="${1#*=}"
            shift
            ;;
        --head-sha=*)
            HEAD_SHA="${1#*=}"
            shift
            ;;
        --description=*)
            DESCRIPTION="${1#*=}"
            shift
            ;;
        --plan-file=*)
            PLAN_FILE="${1#*=}"
            shift
            ;;
        --prompt=*)
            PROMPT="${1#*=}"
            shift
            ;;
        --context=*)
            CONTEXT="${1#*=}"
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Error: Unknown argument: $1" >&2
            echo "" >&2
            usage >&2
            exit 1
            ;;
    esac
done

#############################################
# Validation
#############################################

# Validate mode is provided
if [[ -z "$MODE" ]]; then
    echo "Error: --mode is required" >&2
    echo "" >&2
    usage >&2
    exit 1
fi

# Validate mode value
if [[ "$MODE" != "code-review" && "$MODE" != "general-prompt" ]]; then
    echo "Error: Invalid mode '$MODE'. Must be 'code-review' or 'general-prompt'" >&2
    echo "" >&2
    usage >&2
    exit 1
fi

# Validate mode-specific required arguments
if [[ "$MODE" == "code-review" ]]; then
    if [[ -z "$BASE_SHA" ]]; then
        echo "Error: --base-sha is required for code-review mode" >&2
        echo "" >&2
        usage >&2
        exit 1
    fi

    if [[ -z "$HEAD_SHA" ]]; then
        echo "Error: --head-sha is required for code-review mode" >&2
        echo "" >&2
        usage >&2
        exit 1
    fi

    if [[ -z "$DESCRIPTION" ]]; then
        echo "Error: --description is required for code-review mode" >&2
        echo "" >&2
        usage >&2
        exit 1
    fi
elif [[ "$MODE" == "general-prompt" ]]; then
    if [[ -z "$PROMPT" ]]; then
        echo "Error: --prompt is required for general-prompt mode" >&2
        echo "" >&2
        usage >&2
        exit 1
    fi
fi

#############################################
# Dry Run Exit
#############################################

if [[ "$DRY_RUN" == true ]]; then
    echo "Dry run: Arguments validated successfully"
    echo "Mode: $MODE"
    if [[ "$MODE" == "code-review" ]]; then
        echo "Base SHA: $BASE_SHA"
        echo "Head SHA: $HEAD_SHA"
        echo "Description: $DESCRIPTION"
        [[ -n "$PLAN_FILE" ]] && echo "Plan file: $PLAN_FILE"
    else
        echo "Prompt: $PROMPT"
        [[ -n "$CONTEXT" ]] && echo "Context: (provided)"
    fi
    exit 0
fi

#############################################
# Stage 1: Helper Functions
#############################################

# Context size limit (10KB = 10240 bytes)
CONTEXT_SIZE_LIMIT=10240

# Check context size and warn if >10KB
check_context_size() {
    local context="$1"
    local size=${#context}

    if [[ $size -gt $CONTEXT_SIZE_LIMIT ]]; then
        echo "Warning: Context size is ${size} bytes (>10KB). Consider truncating." >&2
    fi
}

# Build Stage 1 prompt for code review mode
build_code_review_prompt() {
    local base_sha="$1"
    local head_sha="$2"
    local description="$3"
    local plan_file="$4"

    # Get diff
    local diff_output
    diff_output=$(git diff "$base_sha" "$head_sha" 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to get git diff for $base_sha..$head_sha" >&2
        return 1
    fi

    # Get modified files
    local modified_files=$(git diff --name-only "$base_sha" "$head_sha" 2>&1)

    # Load plan if provided
    local plan_content=""
    if [[ -n "$plan_file" && -f "$plan_file" ]]; then
        plan_content=$(cat "$plan_file")
    fi

    # Build context
    local context="# Code Review - Stage 1 Independent Analysis

**Your Task:** Independently review these code changes and provide your analysis.

**Change Description:** $description

**Commits:** $base_sha..$head_sha

**Modified Files:**
$modified_files

"

    if [[ -n "$plan_content" ]]; then
        context+="**Implementation Plan:**
$plan_content

"
    fi

    context+="**Diff:**
\`\`\`diff
$diff_output
\`\`\`

**Instructions:**
Please provide your independent code review in the following format:

## Critical Issues
- [List critical issues, or write 'None']

## Important Issues
- [List important issues, or write 'None']

## Suggestions
- [List suggestions, or write 'None']

Focus on correctness, security, performance, and adherence to the plan (if provided).
"

    echo "$context"
}

# Build Stage 1 prompt for general prompt mode
build_general_prompt_prompt() {
    local prompt="$1"
    local context="$2"

    local full_prompt="# General Analysis - Stage 1 Independent Analysis

**Your Task:** Independently analyze this question and provide your perspective.

**Question:**
$prompt

"

    if [[ -n "$context" ]]; then
        full_prompt+="**Context:**
$context

"
    fi

    full_prompt+="**Instructions:**
Please provide your independent analysis in the following format:

## Strong Points
- [List strong arguments/points, or write 'None']

## Moderate Points
- [List moderate arguments/points, or write 'None']

## Weak Points / Concerns
- [List weak points or concerns, or write 'None']

Provide thoughtful, independent analysis.
"

    echo "$full_prompt"
}

# Run Claude agent
run_claude() {
    local prompt="$1"
    local output_file="$2"

    # For now, Claude is called via a placeholder
    # In a real implementation, this would use Claude API or CLI
    # Since this script is meant to be called BY Claude Code,
    # we'll create a mock response for testing
    cat > "$output_file" <<'EOF'
# Claude Analysis

## Critical Issues
- None

## Important Issues
- None

## Suggestions
- Consider adding more test coverage
EOF

    return 0
}

# Run Gemini agent
run_gemini() {
    local prompt="$1"
    local output_file="$2"

    # For testing: Use mock response instead of real Gemini
    # TODO: Re-enable real Gemini calls in production
    cat > "$output_file" <<'EOF'
GEMINI_NOT_AVAILABLE
EOF
    return 1

    # Real implementation (commented out for testing):
    # Check if gemini CLI is available
    # if ! command -v gemini &> /dev/null; then
    #     echo "GEMINI_NOT_AVAILABLE" > "$output_file"
    #     return 1
    # fi
    #
    # # Call Gemini CLI with timeout
    # timeout 30s gemini "$prompt" > "$output_file" 2>&1
    # local exit_code=$?
    #
    # if [[ $exit_code -eq 124 ]]; then
    #     echo "GEMINI_TIMEOUT" > "$output_file"
    #     return 1
    # fi
    #
    # return $exit_code
}

# Run Codex agent (via MCP)
run_codex() {
    local prompt="$1"
    local output_file="$2"

    # Codex is only available via MCP from within Claude Code
    # This script will be called BY Claude Code, which should handle MCP calls
    # For now, we'll create a placeholder instruction
    cat > "$output_file" <<EOF
CODEX_MCP_REQUIRED

Note: Codex requires MCP call from Claude Code assistant.
The assistant should invoke: mcp__codex-cli__codex with prompt.
EOF

    return 1  # Mark as unavailable for now
}

# Execute Stage 1: Parallel agent execution
execute_stage1() {
    local prompt="$1"

    echo "Stage 1: Launching parallel agent analysis..." >&2

    # Check context size
    check_context_size "$prompt"

    # Create temp files for agent outputs
    local claude_output=$(mktemp)
    local gemini_output=$(mktemp)
    local codex_output=$(mktemp)

    # Track PIDs for background processes
    local claude_pid=""
    local gemini_pid=""
    local codex_pid=""

    # Launch agents in parallel (background)
    # Note: Wrap in subshells to ensure proper backgrounding
    ( run_claude "$prompt" "$claude_output" ) &
    claude_pid=$!

    ( run_gemini "$prompt" "$gemini_output" ) &
    gemini_pid=$!

    ( run_codex "$prompt" "$codex_output" ) &
    codex_pid=$!

    # Wait for all agents (they should complete quickly)
    echo "  Waiting for agents..." >&2

    wait $claude_pid 2>/dev/null || true
    local claude_exit=$?

    wait $gemini_pid 2>/dev/null || true
    local gemini_exit=$?

    wait $codex_pid 2>/dev/null || true
    local codex_exit=$?

    # Read agent responses
    local claude_response=$(cat "$claude_output" 2>/dev/null || echo "")
    local gemini_response=$(cat "$gemini_output" 2>/dev/null || echo "")
    local codex_response=$(cat "$codex_output" 2>/dev/null || echo "")

    # Track success/failure
    local agents_succeeded=0
    local claude_status="failed"
    local gemini_status="failed"
    local codex_status="failed"

    if [[ $claude_exit -eq 0 ]] && [[ -n "$claude_response" ]]; then
        claude_status="success"
        agents_succeeded=$((agents_succeeded + 1))
        echo "  Claude: SUCCESS" >&2
    else
        echo "  Claude: FAILED" >&2
    fi

    if [[ $gemini_exit -eq 0 ]] && [[ -n "$gemini_response" ]] && ! echo "$gemini_response" | grep -q "GEMINI_NOT_AVAILABLE\|GEMINI_TIMEOUT"; then
        gemini_status="success"
        agents_succeeded=$((agents_succeeded + 1))
        echo "  Gemini: SUCCESS" >&2
    else
        if echo "$gemini_response" | grep -q "GEMINI_NOT_AVAILABLE"; then
            echo "  Gemini: NOT AVAILABLE" >&2
        elif echo "$gemini_response" | grep -q "GEMINI_TIMEOUT"; then
            echo "  Gemini: TIMEOUT" >&2
        else
            echo "  Gemini: FAILED" >&2
        fi
    fi

    if [[ $codex_exit -eq 0 ]] && [[ -n "$codex_response" ]] && ! echo "$codex_response" | grep -q "CODEX_MCP_REQUIRED"; then
        codex_status="success"
        agents_succeeded=$((agents_succeeded + 1))
        echo "  Codex: SUCCESS" >&2
    else
        if echo "$codex_response" | grep -q "CODEX_MCP_REQUIRED"; then
            echo "  Codex: MCP REQUIRED (not available in bash)" >&2
        else
            echo "  Codex: FAILED" >&2
        fi
    fi

    echo "  Agents completed: $agents_succeeded/3 succeeded" >&2

    # Cleanup temp files
    rm -f "$claude_output" "$gemini_output" "$codex_output"

    # Check if at least one agent succeeded
    if [[ $agents_succeeded -eq 0 ]]; then
        echo "Error: All agents failed (0/3 succeeded)" >&2
        return 1
    fi

    # Export results for Stage 2 (stored in global variables for now)
    STAGE1_CLAUDE_RESPONSE="$claude_response"
    STAGE1_GEMINI_RESPONSE="$gemini_response"
    STAGE1_CODEX_RESPONSE="$codex_response"
    STAGE1_CLAUDE_STATUS="$claude_status"
    STAGE1_GEMINI_STATUS="$gemini_status"
    STAGE1_CODEX_STATUS="$codex_status"
    STAGE1_AGENTS_SUCCEEDED=$agents_succeeded

    return 0
}

#############################################
# Main Execution
#############################################

# Build Stage 1 prompt based on mode
echo "Building Stage 1 prompt for mode: $MODE" >&2

STAGE1_PROMPT=""

if [[ "$MODE" == "code-review" ]]; then
    STAGE1_PROMPT=$(build_code_review_prompt "$BASE_SHA" "$HEAD_SHA" "$DESCRIPTION" "$PLAN_FILE")
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to build code review prompt" >&2
        exit 1
    fi
elif [[ "$MODE" == "general-prompt" ]]; then
    STAGE1_PROMPT=$(build_general_prompt_prompt "$PROMPT" "$CONTEXT")
fi

# Execute Stage 1
execute_stage1 "$STAGE1_PROMPT"
if [[ $? -ne 0 ]]; then
    echo "Error: Stage 1 failed" >&2
    exit 1
fi

# Stage 2 will be implemented in Task 3
echo "" >&2
echo "Stage 1 complete. Stage 2 (chairman synthesis) not yet implemented." >&2
echo "Agents succeeded: $STAGE1_AGENTS_SUCCEEDED/3" >&2

# For now, just output Stage 1 results
echo ""
echo "# Stage 1 Results"
echo ""
if [[ "$STAGE1_CLAUDE_STATUS" == "success" ]]; then
    echo "## Claude Response"
    echo "$STAGE1_CLAUDE_RESPONSE"
    echo ""
fi

if [[ "$STAGE1_GEMINI_STATUS" == "success" ]]; then
    echo "## Gemini Response"
    echo "$STAGE1_GEMINI_RESPONSE"
    echo ""
fi

if [[ "$STAGE1_CODEX_STATUS" == "success" ]]; then
    echo "## Codex Response"
    echo "$STAGE1_CODEX_RESPONSE"
    echo ""
fi

exit 0
