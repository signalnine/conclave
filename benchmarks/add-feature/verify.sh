#!/usr/bin/env bash
set -euo pipefail

# Check that reverse function exists in src/utils.ts
if ! grep -q "reverse" src/utils.ts; then
  echo "FAIL: reverse function not found in src/utils.ts"
  exit 1
fi

# Check that test files exist
if ! find . -name "*.test.*" -o -name "*.spec.*" | grep -q .; then
  echo "FAIL: no test files found"
  exit 1
fi

# Run tests
npm test
echo "PASS: all checks passed"
