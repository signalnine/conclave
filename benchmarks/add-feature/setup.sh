#!/usr/bin/env bash
set -euo pipefail

# Initialize a Node.js project with vitest for the add-feature benchmark
git init
git config user.email "bench@test"
git config user.name "Bench"

cat > package.json <<'EOF'
{
  "name": "add-feature-bench",
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

mkdir -p src
cat > src/utils.ts <<'EOF'
// Utility functions
export function greet(name: string): string {
  return `Hello, ${name}!`;
}
EOF

npm install --silent 2>/dev/null || true

git add -A
git commit -m "initial scaffold"
