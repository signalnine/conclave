#!/usr/bin/env bash
set -euo pipefail

# Initialize a Node.js project for the marathon key-value store task
git init
git config user.email "bench@test"
git config user.name "Bench"

cat > package.json <<'EOF'
{
  "name": "marathon-bench",
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

npm install --silent 2>/dev/null || true

git add -A
git commit -m "initial scaffold"
