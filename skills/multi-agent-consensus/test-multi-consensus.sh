#!/usr/bin/env bash
set -e

SCRIPT="$(cd "$(dirname "$0")" && pwd)/multi-consensus.sh"

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

# Test: Code review mode missing --base-sha
if $SCRIPT --mode=code-review 2>&1 | grep -q "Error.*base-sha"; then
    echo "✓ Code review mode requires --base-sha"
else
    echo "✗ Code review mode should require --base-sha"
    exit 1
fi

# Test: General prompt mode missing --prompt
if $SCRIPT --mode=general-prompt 2>&1 | grep -q "Error.*prompt"; then
    echo "✓ General prompt mode requires --prompt"
else
    echo "✗ General prompt mode should require --prompt"
    exit 1
fi

# Test: Code review mode extracts git context
test_dir=$(mktemp -d)
script_abs_path="$(cd "$(dirname "$0")" && pwd)/$(basename "$SCRIPT")"
cd "$test_dir"
git init -q
git config user.email "test@test.com"
git config user.name "Test"
echo "original" > file.txt
git add file.txt
git commit -q -m "initial"
base_sha=$(git rev-parse HEAD)
echo "modified" > file.txt
git add file.txt
git commit -q -m "change"
head_sha=$(git rev-parse HEAD)

output=$("$script_abs_path" --mode=code-review --base-sha="$base_sha" --head-sha="$head_sha" --description="test" --dry-run 2>&1 || true)

cd /tmp
rm -rf "$test_dir"

if echo "$output" | grep -q "file.txt"; then
    echo "✓ Code review mode extracts git context"
else
    echo "✗ Code review mode should extract git context"
    exit 1
fi

# Test: General prompt mode includes prompt
output=$("$SCRIPT" --mode=general-prompt --prompt="test question" --context="background" --dry-run 2>&1 || true)

if echo "$output" | grep -q "test question"; then
    echo "✓ General prompt mode includes prompt"
else
    echo "✗ General prompt mode should include prompt"
    exit 1
fi

# Test: Context parameter passes through
if echo "$output" | grep -q "background"; then
    echo "✓ Context parameter passes through"
else
    echo "✗ Context parameter should pass through"
    exit 1
fi

# Test: Severity labels based on mode
# Note: Full verification requires running actual reviewers, tested in integration

echo "All tests passed!"
