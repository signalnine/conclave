#!/usr/bin/env bash
set -euo pipefail

# Initialize a Node.js project with a buggy capitalize function
git init
git config user.email "bench@test"
git config user.name "Bench"

cat > package.json <<'EOF'
{
  "name": "fix-bug-bench",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "test": "npx vitest run"
  },
  "devDependencies": {
    "vitest": "^1.0.0"
  }
}
EOF

cat > vitest.config.ts <<'EOF'
import { defineConfig } from 'vitest/config'
export default defineConfig({
  test: { globals: true }
})
EOF

mkdir -p src __tests__

# Buggy capitalize: throws on empty string
cat > src/utils.ts <<'EOF'
export function capitalize(s: string): string {
  return s[0].toUpperCase() + s.slice(1);
}
EOF

# Existing tests that pass (don't test empty string)
cat > __tests__/utils.test.ts <<'EOF'
import { describe, it, expect } from 'vitest'
import { capitalize } from '../src/utils'

describe('capitalize', () => {
  it('capitalizes first letter', () => {
    expect(capitalize('hello')).toBe('Hello')
  })

  it('handles single character', () => {
    expect(capitalize('a')).toBe('A')
  })
})
EOF

npm install --silent 2>/dev/null || true

git add -A
git commit -m "initial scaffold with buggy capitalize"
