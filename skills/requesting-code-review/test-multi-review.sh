#!/usr/bin/env bash
set -e

echo "Testing multi-review.sh..."

# Test: Missing arguments
if ./skills/requesting-code-review/multi-review.sh 2>&1 | grep -q "Usage:"; then
    echo "✓ Shows usage when no arguments"
else
    echo "✗ Should show usage when no arguments"
    exit 1
fi

# Test: Context preparation
# Create a test repo with commits
test_dir=$(mktemp -d)
cd "$test_dir"
git init -q
echo "line1" > file.txt
git add file.txt
git commit -q -m "first commit"
FIRST_SHA=$(git rev-parse HEAD)
echo "line2" >> file.txt
git commit -q -am "second commit"
SECOND_SHA=$(git rev-parse HEAD)

# Run script in dry-run mode (add --dry-run flag)
if $OLDPWD/skills/requesting-code-review/multi-review.sh \
    "$FIRST_SHA" "$SECOND_SHA" "-" "test change" --dry-run 2>&1 \
    | grep -q "Modified files: 1"; then
    echo "✓ Extracts git context correctly"
else
    echo "✗ Should extract git context"
    exit 1
fi

cd "$OLDPWD"

# Test: Empty diff (same commit)
cd "$test_dir"
SAME_SHA=$(git rev-parse HEAD)
if $OLDPWD/skills/requesting-code-review/multi-review.sh \
    "$SAME_SHA" "$SAME_SHA" "-" "no changes" --dry-run 2>&1 \
    | grep -q "Modified files: 0"; then
    echo "✓ Handles empty diff correctly"
else
    echo "✗ Should report 0 files for empty diff"
    exit 1
fi
cd "$OLDPWD"

rm -rf "$test_dir"

# Test: Issue similarity matching
echo "Testing consensus algorithm..."

# Source the helper functions from multi-review.sh. Run in a subshell so its
# set -euo pipefail and globals do not leak into this script.
(
    source ./skills/requesting-code-review/multi-review.sh

    test_filename_extraction() {
        # Test path normalization
        file1=$(extract_filename "Error in ./src/foo.js line 10")
        file2=$(extract_filename "Error in src/foo.js line 10")
        if [ "$file1" = "$file2" ] && [ "$file1" = "src/foo.js" ]; then
            echo "✓ Filename normalization works"
        else
            echo "✗ Filename normalization failed: '$file1' != '$file2'"
            exit 1
        fi
    }

    test_word_overlap() {
        # Test similar descriptions
        overlap=$(word_overlap_percent "missing error handling in main function" "no error handling in main function")
        if [ "$overlap" -ge 60 ]; then
            echo "✓ Word overlap detects similar issues ($overlap%)"
        else
            echo "✗ Word overlap too low for similar issues: $overlap%"
            exit 1
        fi

        # Test dissimilar descriptions
        overlap=$(word_overlap_percent "missing error handling" "add logging support")
        if [ "$overlap" -lt 60 ]; then
            echo "✓ Word overlap rejects dissimilar issues ($overlap%)"
        else
            echo "✗ Word overlap too high for dissimilar issues: $overlap%"
            exit 1
        fi
    }

    test_issue_similarity() {
        # Same file, similar content
        if issues_similar "Missing validation in src/foo.js" "No validation in src/foo.js"; then
            echo "✓ Similar issues in same file matched"
        else
            echo "✗ Should match similar issues in same file"
            exit 1
        fi

        # Different files
        if ! issues_similar "Missing validation in src/foo.js" "Missing validation in src/bar.js"; then
            echo "✓ Issues in different files not matched"
        else
            echo "✗ Should not match issues in different files"
            exit 1
        fi

        # Same file, different content
        if ! issues_similar "Missing input validation" "Need better documentation"; then
            echo "✓ Dissimilar issues not matched"
        else
            echo "✗ Should not match dissimilar issues"
            exit 1
        fi
    }

    test_filename_extraction
    test_word_overlap
    test_issue_similarity
)

echo "All tests passed!"
