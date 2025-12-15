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
# Main Execution (Placeholder for Task 2+)
#############################################

echo "consensus-synthesis.sh: Implementation in progress"
echo "Mode: $MODE"
echo "Stage 1 and Stage 2 execution will be implemented in subsequent tasks"
exit 0
