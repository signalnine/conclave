#!/usr/bin/env bash
set -euo pipefail

# Check that key-value store exists with required operations
for op in get set delete list; do
  if ! grep -rq "$op" src/; then
    echo "FAIL: $op operation not found in src/"
    exit 1
  fi
done

# Check TTL support
if ! grep -rqi "ttl\|expir" src/; then
  echo "FAIL: TTL support not found"
  exit 1
fi

# Check namespace support
if ! grep -rqi "namespace\|prefix" src/; then
  echo "FAIL: namespace/prefix support not found"
  exit 1
fi

# Run tests
npm test
echo "PASS: all checks passed"
