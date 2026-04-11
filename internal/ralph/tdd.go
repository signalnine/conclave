package ralph

// TDDPreamble is prepended to every ralph-loop iteration prompt to enforce
// the v10 methodology: Contract -> TDD -> Boil the Lake -> Verify Against Contract.
// Validated at 90.2% in Thunderdome benchmarks (7,500+ trials).
const TDDPreamble = `## MANDATORY: How to Work

### 1. Understand First
Read the task fully. Read existing code, tests, and config files. Understand what exists before writing anything.

### 2. Write a Contract BEFORE Any Code
Before writing any implementation, create a CONTRACT.md file that defines:
- Every behavior the finished code must exhibit -- be specific and exhaustive
- How to verify each behavior -- the exact test, command, or check that proves it works
- What done looks like for each criterion

### 3. Test-First Development (MANDATORY)
For each contract criterion, you MUST write a failing test BEFORE any implementation code.

**The process:**
1. Pick the next contract criterion
2. Write a test that verifies it
3. Run it -- watch it FAIL (this proves the test works)
4. Write the minimal code to make it pass
5. Run it -- watch it PASS
6. Repeat for the next criterion

Wrote code before a test? Delete it. Start over with a failing test.

### 4. Boil the Lake
Handle ALL edge cases, not just happy paths. Write comprehensive tests -- cover boundaries, errors, empty inputs. Implement the full feature, not 90% of it.

### 5. Verify Against Contract
Go through CONTRACT.md line by line. Run each verification check. Fix ALL failures. Do not stop until every criterion passes, tests pass, build is clean, lint is clean.

## MANDATORY: Completion Gate

Before claiming done, you MUST:
1. Run the FULL project verification suite (tests + build + lint), not just your new tests
2. Read COMPLETE output. Count failures.
3. If ANY failure: fix and re-run. Do NOT claim done.
4. Commit your work
5. Review your diff: git diff HEAD~1
6. Look for: missing edge cases, incomplete implementations, dead code, debug artifacts
7. If issues found: fix, re-verify, re-commit
8. Only report success when verification passes AND diff review is clean
`
