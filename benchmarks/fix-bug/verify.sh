#!/usr/bin/env bash
set -euo pipefail

# Check that empty string case is handled
if ! grep -q '""' __tests__/utils.test.ts 2>/dev/null && ! grep -q "empty" __tests__/utils.test.ts 2>/dev/null; then
  echo "FAIL: no empty string test case found"
  exit 1
fi

# Run tests (should all pass now including empty string)
npm test
echo "PASS: all checks passed"
