#!/usr/bin/env bash
set -e

SCRIPT="$(cd "$(dirname "$0")" && pwd)/consensus-synthesis.sh"

echo "Testing consensus-synthesis.sh..."

# Test 1: Missing --mode flag
echo -n "Test 1: Requires --mode flag... "
if $SCRIPT 2>&1 | grep -q "Error.*--mode"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about --mode"
    exit 1
fi

# Test 2: Invalid mode value
echo -n "Test 2: Rejects invalid mode... "
if $SCRIPT --mode=invalid 2>&1 | grep -q "Error.*Invalid mode"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about invalid mode"
    exit 1
fi

# Test 3: Code review mode - missing --base-sha
echo -n "Test 3: Code review requires --base-sha... "
if $SCRIPT --mode=code-review 2>&1 | grep -q "Error.*base-sha"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about missing --base-sha"
    exit 1
fi

# Test 4: Code review mode - missing --head-sha
echo -n "Test 4: Code review requires --head-sha... "
if $SCRIPT --mode=code-review --base-sha=abc123 2>&1 | grep -q "Error.*head-sha"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about missing --head-sha"
    exit 1
fi

# Test 5: Code review mode - missing --description
echo -n "Test 5: Code review requires --description... "
if $SCRIPT --mode=code-review --base-sha=abc123 --head-sha=def456 2>&1 | grep -q "Error.*description"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about missing --description"
    exit 1
fi

# Test 6: General prompt mode - missing --prompt
echo -n "Test 6: General prompt requires --prompt... "
if $SCRIPT --mode=general-prompt 2>&1 | grep -q "Error.*prompt"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Error message about missing --prompt"
    exit 1
fi

# Test 7: Help/usage message
echo -n "Test 7: Shows usage with --help... "
if $SCRIPT --help 2>&1 | grep -q "Usage:"; then
    echo "PASS"
else
    echo "FAIL"
    echo "  Expected: Usage message"
    exit 1
fi

# Test 8: Valid code review arguments (dry-run check)
echo -n "Test 8: Accepts valid code review arguments... "
# Using a dry-run flag to avoid actual execution
output=$($SCRIPT --mode=code-review --base-sha=abc123 --head-sha=def456 --description="test change" --dry-run 2>&1 || true)
if echo "$output" | grep -q "Error"; then
    echo "FAIL"
    echo "  Expected: No error with valid arguments"
    echo "  Got: $output"
    exit 1
else
    echo "PASS"
fi

# Test 9: Valid general prompt arguments (dry-run check)
echo -n "Test 9: Accepts valid general prompt arguments... "
output=$($SCRIPT --mode=general-prompt --prompt="test question" --dry-run 2>&1 || true)
if echo "$output" | grep -q "Error"; then
    echo "FAIL"
    echo "  Expected: No error with valid arguments"
    echo "  Got: $output"
    exit 1
else
    echo "PASS"
fi

# Test 10: Code review with optional --plan-file
echo -n "Test 10: Accepts optional --plan-file... "
output=$($SCRIPT --mode=code-review --base-sha=abc123 --head-sha=def456 --description="test" --plan-file=/tmp/plan.md --dry-run 2>&1 || true)
if echo "$output" | grep -q "Error"; then
    echo "FAIL"
    echo "  Expected: No error with optional --plan-file"
    echo "  Got: $output"
    exit 1
else
    echo "PASS"
fi

# Test 11: General prompt with optional --context
echo -n "Test 11: Accepts optional --context... "
output=$($SCRIPT --mode=general-prompt --prompt="test" --context="background info" --dry-run 2>&1 || true)
if echo "$output" | grep -q "Error"; then
    echo "FAIL"
    echo "  Expected: No error with optional --context"
    echo "  Got: $output"
    exit 1
else
    echo "PASS"
fi

echo ""
echo "All argument parsing tests passed!"
