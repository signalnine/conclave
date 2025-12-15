# Multi-Agent Consensus Framework

Reusable infrastructure for multi-agent consensus. Any skill can invoke Claude, Gemini, and Codex to get diverse perspectives on prompts, designs, code, or decisions.

## Purpose

Different AI models have different strengths and weaknesses. Single agents may miss issues, exhibit biases, or have blind spots. This framework provides consensus from multiple agents, grouped by agreement level.

## Architecture

**Design:** `docs/plans/2025-12-13-multi-agent-consensus-framework-design.md`

**Key components:**
- Mode-based interface (code-review vs general-prompt)
- Shared consensus algorithm (word overlap + file matching)
- Three-tier output (High/Medium/Consider priority)
- Graceful degradation (works with 1, 2, or 3 reviewers)

## Setup

### Required (Always Needed)

**1. Bash 4.0+**
```bash
# Check version
bash --version

# macOS: upgrade if needed
brew install bash
```

**2. bc (calculator)**
```bash
# Check if installed
which bc

# Install if needed
# macOS:
brew install bc

# Debian/Ubuntu:
sudo apt-get install bc
```

**3. git**
```bash
# Check if installed
git --version
```

**4. Claude Code**

You're already using it! Claude is the only *required* reviewer. Gemini and Codex are optional.

### Optional Reviewers

**5. Gemini CLI (Optional but Recommended)**

Enables Gemini reviews for more diverse perspectives:

```bash
# Install Google's Gemini CLI
npm install -g @google/generative-ai-cli
# or
pip install google-generativeai-cli

# Verify installation
gemini --version

# Set up API key (get from https://ai.google.dev/)
export GEMINI_API_KEY="your-api-key-here"

# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
echo 'export GEMINI_API_KEY="your-api-key-here"' >> ~/.bashrc
```

**Without Gemini:** Framework still works with Claude + Codex (or just Claude alone).

**6. Codex MCP (Optional but Recommended)**

Enables Codex reviews via Claude Code's MCP integration:

```bash
# Check if installed
claude-code mcp list | grep codex

# If not installed, add codex-cli MCP server
# (Follow Claude Code MCP setup instructions)
```

**Without Codex:** Framework still works with Claude + Gemini (or just Claude alone).

## Verification

Test your setup:

```bash
# Test basic functionality (uses Claude only)
./skills/multi-agent-consensus/test-multi-consensus.sh

# Test with actual reviewers
echo "test" > /tmp/test.txt
git init /tmp/test-repo
cd /tmp/test-repo
git add test.txt
git commit -m "initial"
BASE=$(git rev-parse HEAD)
echo "modified" > test.txt
git add test.txt
git commit -m "change"
HEAD=$(git rev-parse HEAD)

# This will show which reviewers are available
../path/to/skills/multi-agent-consensus/multi-consensus.sh --mode=code-review \
  --base-sha="$BASE" --head-sha="$HEAD" --description="test"

# Look for:
# Claude: ✓ (always works)
# Gemini: ✓ or ✗ (not installed)
# Codex: ✓ or ✗ (not available)
```

## Usage

### Code Review Mode

```bash
skills/multi-agent-consensus/multi-consensus.sh --mode=code-review \
  --base-sha="abc123" \
  --head-sha="def456" \
  --plan-file="docs/plans/feature.md" \
  --description="Add authentication"
```

### General Prompt Mode

```bash
skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="What could go wrong with this design?" \
  --context="$(cat design.md)"
```

## Output Format

Three-tier consensus report:

```markdown
## High Priority - All Reviewers Agree
- [SEVERITY] description
  - Claude: "issue text"
  - Gemini: "issue text"
  - Codex: "issue text"

## Medium Priority - Majority Flagged (2/3)
- [SEVERITY] description
  - Claude: "issue text"
  - Gemini: "issue text"

## Consider - Single Reviewer Mentioned
- [SEVERITY] description
  - Codex: "issue text"
```

## Configuration

- `SIMILARITY_THRESHOLD=60` - Word overlap threshold for matching issues (default 60%)

## Dependencies

- Bash 4.0+
- git
- bc (for calculations)
- gemini CLI (optional, for Gemini reviews)
- Claude Code (optional, for Claude/Codex reviews)

## Integration Examples

### Already Integrated

**1. Brainstorming (design validation)**

After design approval, offers multi-agent validation:
```bash
DESIGN=$(cat docs/plans/2025-12-13-feature-design.md)

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Review this design for architectural flaws, over-engineering, missing requirements, maintainability concerns, or testing gaps. Rate as STRONG/MODERATE/WEAK." \
  --context="$DESIGN"
```

**2. Requesting Code Review**

Automatically uses consensus framework:
```bash
skills/multi-agent-consensus/multi-consensus.sh --mode=code-review \
  --base-sha="abc123" --head-sha="def456" \
  --description="Add authentication feature"
```

### Ready to Integrate

**3. Architecture Decisions**

Get consensus on technical choices:
```bash
skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Which approach is better for this use case and why? Rate confidence as STRONG/MODERATE/WEAK." \
  --context="Option A: Redis caching. Option B: In-memory caching. Use case: 1000 req/sec API with 5-minute session TTL."
```

**4. Debugging (root cause analysis)**

Multiple perspectives on error causes:
```bash
ERROR_CONTEXT="Stack trace shows null pointer in database connection pool. Happens randomly under load."

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="What could cause this error? List potential root causes. Rate likelihood as STRONG/MODERATE/WEAK." \
  --context="$ERROR_CONTEXT"
```

**5. Security Review**

Consensus on security concerns:
```bash
CODE=$(cat src/auth/login.py)

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Identify security vulnerabilities in this authentication code. Rate severity as STRONG/MODERATE/WEAK." \
  --context="$CODE"
```

**6. Performance Optimization**

Get diverse perspectives on bottlenecks:
```bash
PROFILE_DATA=$(cat profiling-results.txt)

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Analyze this performance profile. What are the bottlenecks and how should they be addressed? Rate impact as STRONG/MODERATE/WEAK." \
  --context="$PROFILE_DATA"
```

**7. API Design Review**

Consensus on interface design:
```bash
API_SPEC=$(cat openapi.yaml)

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Review this API design for usability issues, inconsistencies, or missing endpoints. Rate importance as STRONG/MODERATE/WEAK." \
  --context="$API_SPEC"
```

**8. Refactoring Decisions**

Should you refactor and how:
```bash
LEGACY_CODE=$(cat legacy-module.js)

skills/multi-agent-consensus/multi-consensus.sh --mode=general-prompt \
  --prompt="Should this code be refactored? If yes, what approach? Rate urgency as STRONG/MODERATE/WEAK." \
  --context="$LEGACY_CODE"
```

## Testing

```bash
./skills/multi-agent-consensus/test-multi-consensus.sh
```

## Migration from multi-review.sh

Old code review calls:
```bash
skills/requesting-code-review/multi-review.sh "$BASE" "$HEAD" "$PLAN" "$DESC"
```

New code review calls:
```bash
skills/multi-agent-consensus/multi-consensus.sh --mode=code-review \
  --base-sha="$BASE" --head-sha="$HEAD" \
  --plan-file="$PLAN" --description="$DESC"
```
