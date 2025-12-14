#!/usr/bin/env bash
set -e

SCRIPT="$(dirname "$0")/multi-consensus.sh"

echo "Testing multi-consensus.sh..."

# Test: Missing --mode flag
if $SCRIPT 2>&1 | grep -q "Error.*--mode"; then
    echo "✓ Requires --mode flag"
else
    echo "✗ Should require --mode flag"
    exit 1
fi

# Test: Invalid mode value
if $SCRIPT --mode=invalid 2>&1 | grep -q "Error.*Invalid mode"; then
    echo "✓ Rejects invalid mode"
else
    echo "✗ Should reject invalid mode"
    exit 1
fi

# Test: Valid mode without required args
if $SCRIPT --mode=code-review 2>&1 | grep -q "Error.*base-sha"; then
    echo "✓ Code review mode requires --base-sha"
else
    echo "✗ Code review mode should require --base-sha"
    exit 1
fi

if $SCRIPT --mode=general-prompt 2>&1 | grep -q "Error.*prompt"; then
    echo "✓ General prompt mode requires --prompt"
else
    echo "✗ General prompt mode should require --prompt"
    exit 1
fi

# Test: Code review mode prepares git context
test_code_review_context() {
    # Create temporary git repo for testing
    local test_dir=$(mktemp -d)
    cd "$test_dir"
    git init -q
    git config user.email "test@test.com"
    git config user.name "Test"
    echo "original" > file.txt
    git add file.txt
    git commit -q -m "initial"
    local base_sha=$(git rev-parse HEAD)
    echo "modified" > file.txt
    git add file.txt
    git commit -q -m "change"
    local head_sha=$(git rev-parse HEAD)

    # Run with --dry-run to test context preparation
    local output=$("$OLDPWD/$SCRIPT" --mode=code-review --base-sha="$base_sha" --head-sha="$head_sha" --description="test" --dry-run 2>&1 || true)

    cd "$OLDPWD"
    rm -rf "$test_dir"

    if echo "$output" | grep -q "file.txt"; then
        echo "✓ Code review mode extracts file changes"
        return 0
    else
        echo "✗ Code review mode should extract file changes"
        echo "Output was: $output"
        return 1
    fi
}

# Test: General prompt mode formats prompt
test_general_prompt_context() {
    local output=$("$SCRIPT" --mode=general-prompt --prompt="test question" --context="background" --dry-run 2>&1 || true)

    if echo "$output" | grep -q "test question"; then
        echo "✓ General prompt mode includes prompt"
        return 0
    else
        echo "✗ General prompt mode should include prompt"
        echo "Output was: $output"
        return 1
    fi
}

# Run new tests
test_code_review_context
test_general_prompt_context

echo "All tests passed!"
