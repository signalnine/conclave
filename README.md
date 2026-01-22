# Superpowers

Superpowers is a complete software development workflow for your coding agents, built on top of a set of composable "skills" and some initial instructions that make sure your agent uses them.

## This Fork: Multi-Agent Consensus

![The Council](council.png)

**This fork adds automatic multi-agent consensus** - a council of AI reviewers (Claude, Gemini, and Codex) that independently analyze your work, then synthesize their perspectives into prioritized recommendations.

### What's Different From Upstream

| Feature | Upstream | This Fork |
|---------|----------|-----------|
| Code review | Single reviewer | 3 agents + consensus synthesis |
| Design validation | Optional | Automatic after brainstorming |
| Plan validation | None | Architecture/risk review before execution |
| Bug analysis | Single perspective | Multi-agent root cause validation |
| Final verification | Local tests only | Tests + multi-agent review |
| Brainstorming | Interactive only | **Autopilot mode** - consensus answers questions |

### Automatic Consensus Integration

Consensus review triggers automatically at key workflow points:

```
Brainstorming → Writing Plans → Execution → Debugging → Verification
     ↓              ↓              ↓           ↓            ↓
  Design        Architecture    Per-task    Root cause   Final check
  validation    validation      review      validation   before done
```

**7 skills enhanced with consensus:**
- `brainstorming` - Design validation + autopilot mode
- `writing-plans` - Architecture/risk/scope validation
- `subagent-driven-development` - Third review stage after code quality
- `executing-plans` - Consensus review after each batch
- `finishing-a-development-branch` - Final review before merge
- `systematic-debugging` - Root cause hypothesis validation
- `verification-before-completion` - Multi-agent final check

### Consensus Autopilot (New)

In brainstorming, you can now choose **Consensus Autopilot** mode:

```
Two modes available:
1. Interactive - I ask questions, you answer
2. Consensus Autopilot - Multi-agent consensus answers questions,
   you watch and can interrupt anytime to override
```

The council debates each design decision while you watch. Interrupt anytime to override, go back, or take over.

### Using Consensus Directly

```bash
# Review code changes
./skills/multi-agent-consensus/auto-review.sh "Added authentication"

# Review with explicit base
./skills/multi-agent-consensus/auto-review.sh --base=HEAD~5 "Recent fixes"

# General question
./skills/multi-agent-consensus/consensus-synthesis.sh \
  --mode=general-prompt \
  --prompt="What could go wrong with this architecture?" \
  --context="$(cat design.md)"
```

---

## How it works

It starts from the moment you fire up your coding agent. As soon as it sees that you're building something, it *doesn't* just jump into trying to write code. Instead, it steps back and asks you what you're really trying to do. 

Once it's teased a spec out of the conversation, it shows it to you in chunks short enough to actually read and digest. 

After you've signed off on the design, your agent puts together an implementation plan that's clear enough for an enthusiastic junior engineer with poor taste, no judgement, no project context, and an aversion to testing to follow. It emphasizes true red/green TDD, YAGNI (You Aren't Gonna Need It), and DRY. 

Next up, once you say "go", it launches a *subagent-driven-development* process, having agents work through each engineering task, inspecting and reviewing their work, and continuing forward. It's not uncommon for Claude to be able to work autonomously for a couple hours at a time without deviating from the plan you put together.

There's a bunch more to it, but that's the core of the system. And because the skills trigger automatically, you don't need to do anything special. Your coding agent just has Superpowers.


## Sponsorship

If Superpowers has helped you do stuff that makes money and you are so inclined, I'd greatly appreciate it if you'd consider [sponsoring my opensource work](https://github.com/sponsors/obra).

Thanks! 

- Jesse


## Installation

**Note:** Installation differs by platform. Claude Code has a built-in plugin system. Codex and OpenCode require manual setup.

### Claude Code (via Plugin Marketplace)

In Claude Code, register the marketplace first:

```bash
/plugin marketplace add signalnine/honest-gabes-marketplace
```

Then install the plugin from this marketplace:

```bash
/plugin install superpowers@honest-gabes-marketplace
```

> **Note:** This fork is available from [Honest Gabe's Marketplace](https://github.com/signalnine/honest-gabes-marketplace). For the upstream version without multi-agent consensus, use `obra/superpowers-marketplace` instead.

### Verify Installation

Check that commands appear:

```bash
/help
```

```
# Should see:
# /superpowers:brainstorm - Interactive design refinement
# /superpowers:write-plan - Create implementation plan
# /superpowers:execute-plan - Execute plan in batches
```

### Codex

Tell Codex:

```
Fetch and follow instructions from https://raw.githubusercontent.com/obra/superpowers/refs/heads/main/.codex/INSTALL.md
```

**Detailed docs:** [docs/README.codex.md](docs/README.codex.md)

### OpenCode

Tell OpenCode:

```
Fetch and follow instructions from https://raw.githubusercontent.com/obra/superpowers/refs/heads/main/.opencode/INSTALL.md
```

**Detailed docs:** [docs/README.opencode.md](docs/README.opencode.md)

## The Basic Workflow

1. **brainstorming** - Activates before writing code. Refines rough ideas through questions, explores alternatives, presents design in sections for validation. Saves design document.

2. **using-git-worktrees** - Activates after design approval. Creates isolated workspace on new branch, runs project setup, verifies clean test baseline.

3. **writing-plans** - Activates with approved design. Breaks work into bite-sized tasks (2-5 minutes each). Every task has exact file paths, complete code, verification steps.

4. **subagent-driven-development** or **executing-plans** - Activates with plan. Dispatches fresh subagent per task with two-stage review (spec compliance, then code quality), or executes in batches with human checkpoints.

5. **test-driven-development** - Activates during implementation. Enforces RED-GREEN-REFACTOR: write failing test, watch it fail, write minimal code, watch it pass, commit. Deletes code written before tests.

6. **requesting-code-review** - Activates between tasks. Multi-reviewer consensus from Claude, Gemini, and Codex. Groups issues by agreement level (all agree → high priority, majority → medium, single → consider). Critical issues block progress.

7. **finishing-a-development-branch** - Activates when tasks complete. Verifies tests, presents options (merge/PR/keep/discard), cleans up worktree.

**The agent checks for relevant skills before any task.** Mandatory workflows, not suggestions.

## What's Inside

### Skills Library

**Testing**
- **test-driven-development** - RED-GREEN-REFACTOR cycle (includes testing anti-patterns reference)

**Debugging**
- **systematic-debugging** - 4-phase root cause process (includes root-cause-tracing, defense-in-depth, condition-based-waiting techniques)
- **verification-before-completion** - Ensure it's actually fixed

**Collaboration**
- **brainstorming** - Socratic design refinement with optional multi-agent validation
- **multi-agent-consensus** - Get consensus from Claude/Gemini/Codex on any prompt (design validation, architecture decisions, debugging, code review)
- **writing-plans** - Detailed implementation plans
- **executing-plans** - Batch execution with checkpoints
- **dispatching-parallel-agents** - Concurrent subagent workflows
- **requesting-code-review** - Multi-reviewer code review using consensus framework
- **receiving-code-review** - Responding to feedback
- **using-git-worktrees** - Parallel development branches
- **finishing-a-development-branch** - Merge/PR decision workflow
- **subagent-driven-development** - Fast iteration with two-stage review (spec compliance, then code quality)
- **ralph-loop** - Autonomous iteration wrapper - runs tasks until success or iteration cap hit (fresh context per iteration, stuck detection, failure branches)

**Meta**
- **writing-skills** - Create new skills following best practices (includes testing methodology)
- **using-superpowers** - Introduction to the skills system

## Philosophy

- **Test-Driven Development** - Write tests first, always
- **Systematic over ad-hoc** - Process over guessing
- **Complexity reduction** - Simplicity as primary goal
- **Evidence over claims** - Verify before declaring success

Read more: [Superpowers for Claude Code](https://blog.fsck.com/2025/10/09/superpowers/)

## Contributing

Skills live directly in this repository. To contribute:

1. Fork the repository
2. Create a branch for your skill
3. Follow the `writing-skills` skill for creating and testing new skills
4. Submit a PR

See `skills/writing-skills/SKILL.md` for the complete guide.

## Updating

Skills update automatically when you update the plugin:

```bash
/plugin update superpowers
```

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: https://github.com/obra/superpowers/issues
- **Marketplace**: https://github.com/obra/superpowers-marketplace
