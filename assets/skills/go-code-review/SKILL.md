---
name: go-code-review
description: "Perform structured code reviews for Go projects, comparing branch changes against Jira ticket requirements. Creates feedback files in agent-specific directories and optionally comments on PRs. Use when: reviewing code changes, verifying ticket implementation, providing code review feedback."
---

# Go Code Review Skill

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Fetch the Pull Request diff and details using `gh` CLI |
| 2 | Extract Jira ticket ID from PR title/branch |
| 3 | Fetch Jira ticket and verify access |
| 4 | Analyze PR diff against ticket requirements |
| 5 | Write feedback to `./agent-feedback/<agent>/<TICKET>.md` |
| 6 | Comment on PR using `gh` CLI (if possible) |

## CRITICAL: Pre-Flight Checks

**Before starting review, verify:**

### 1. Jira Access
```
Can you access Jira and read the ticket?

If NO:
❌ STOP immediately
✅ Tell user: "Cannot access Jira to fetch ticket <TICKET>. 
             Please provide ticket details manually or grant Jira access."
```

### 2. GitHub CLI Access
```
Can you use the `gh` CLI to fetch PR diffs?

If NO:
❌ STOP immediately
✅ Tell user: "Cannot access the `gh` CLI. 
             Need `gh` installed and authenticated to fetch the PR."
```

### 3. Agent Name
```
Do you know which agent you are? (codex, copilot, antigravity, warp, claude)

If NO or UNCLEAR:
❌ STOP immediately
✅ Ask user: "Which agent am I? (codex/copilot/antigravity/warp/claude)
             Needed for feedback file path."
```

## Workflow

### Step 1: Fetch Pull Request

You are provided the PR number and Repository in the prompt.
Use the `gh` CLI to fetch the PR contents. Do NOT rely on local `git diff` as you may not have the repo cloned locally.

```bash
# 1. Get PR metadata (title, body, base branch, head branch)
gh pr view <PR_NUMBER> --repo <REPO>

# 2. Get the actual code changes
gh pr diff <PR_NUMBER> --repo <REPO>
```

**If `gh` commands fail:**
❌ STOP immediately
✅ Tell user: "Cannot fetch PR <PR_NUMBER> from <REPO>. Please check my GitHub CLI authentication or if the PR exists."

### Step 2: Fetch Jira Ticket

Extract the Jira ticket ID (e.g., TBY-31) from the PR branch name or PR title.

```bash
TICKET="TBY-31"  # Derived from PR
```

**If ticket fetch fails:**
❌ STOP immediately
✅ Tell user: "Cannot fetch ticket <TICKET> from Jira. 
             Options:
             1. Verify ticket ID is correct
             2. Grant me Jira read access
             3. Provide ticket details manually"

### Step 3: Analyze Changes

Compare the fetched PR diff against the Jira ticket requirements using the **Go Code Review Checklist**.

### Step 5: Write Feedback File

**File path format:**
```
agent-feedback/<agent-name>/<TICKET>.md
```

**Example paths:**
- `agent-feedback/codex/TBY-31.md`
- `agent-feedback/claude/DT-156.md`
- `agent-feedback/copilot/TBY-45.md`

**Create directory if needed:**
```bash
mkdir -p agent-feedback/<agent-name>
```

**Feedback file template:**
```markdown
# Code Review: <TICKET>

**Reviewer**: <Agent Name> (e.g., codex, claude, copilot)
**Date**: <YYYY-MM-DD HH:MM>
**Branch**: <branch-name>
**Ticket**: <TICKET-URL>
**PR**: <PR-URL or "No PR found">

---

## Ticket Summary

<Copy ticket summary from Jira>

## Acceptance Criteria

<List acceptance criteria from ticket>

---

## Review Summary

**Overall Assessment**: ✅ Approved / ⚠️ Needs Changes / ❌ Rejected

**Criteria Met**: X/Y

<Brief 2-3 sentence summary>

---

## Detailed Findings

### ✅ What's Working Well

1. **[Category]**: Description
   - Specific example or file reference
   
2. **[Category]**: Description
   - Specific example or file reference

### ⚠️ Issues to Address

#### Critical
1. **[Issue]**: Description
   - **File**: `path/to/file.go:123`
   - **Problem**: What's wrong
   - **Fix**: What should be done
   - **Why**: Why this matters

#### Major
1. **[Issue]**: Description

#### Minor
1. **[Issue]**: Description

### 💡 Suggestions (Optional)

1. **[Enhancement]**: Description

---

## Acceptance Criteria Verification

- [ ] **Criterion 1**: <criterion text>
  - Status: ✅ Met / ❌ Not Met / ⚠️ Partially Met
  - Evidence: <file/commit reference>
  - Notes: <any notes>

- [ ] **Criterion 2**: <criterion text>
  - Status: ✅ Met / ❌ Not Met / ⚠️ Partially Met
  - Evidence: <file/commit reference>

[Repeat for all criteria]

---

## Code Quality Assessment

### Tests
- **Coverage**: <percentage or "Not measured">
- **Test Quality**: <assessment>
- **Missing Tests**: <list any>

### Go Standards Compliance
- **testify usage**: ✅ / ❌
- **Table-driven tests**: ✅ / ❌ / N/A
- **Error handling**: ✅ / ⚠️ / ❌
- **Context usage**: ✅ / ⚠️ / ❌
- **Concurrency (if applicable)**: ✅ / ⚠️ / ❌

### Architecture
- **Package boundaries**: ✅ / ⚠️ / ❌
- **Dependency usage**: ✅ / ⚠️ / ❌
- **Interface design**: ✅ / ⚠️ / ❌ / N/A

---

## Files Changed

<List all files with brief description of changes>

```
pkg/api/handler.go     - Added health check endpoint
pkg/api/handler_test.go - Added tests for health check
docs/api.md            - Updated API documentation
```

---

## Commits in Branch

<List commits>

```
abc1234 TBY-31 feat(api): add health check endpoint
def5678 TBY-31 test(api): add health check tests
```

---

## Recommendations

### Must Fix Before Merge
1. <Critical issue that blocks merge>

### Should Fix Before Merge
1. <Important issue that should be addressed>

### Can Address Later
1. <Minor issue or enhancement that can be follow-up>

---

## Next Steps

<What should happen next? e.g., "Ready to merge after fixing critical issues", "Needs significant rework", etc.>

---

**Review completed by**: <Agent Name>
```

### Step 6: Comment on PR (If Possible)

**Only attempt if:**
- ✅ PR exists
- ✅ You have API access to GitHub/GitLab
- ✅ You have permission to comment

**PR Comment Format:**
```markdown
## Code Review Feedback

Hi! I've completed a code review of this PR against ticket [<TICKET>](<ticket-url>).

### Summary
<Brief 2-3 sentence summary from feedback file>

### Assessment
**Overall**: ✅ Approved / ⚠️ Needs Changes / ❌ Needs Major Rework

**Acceptance Criteria**: X/Y met

### Key Findings

#### Critical Issues
<List critical issues if any>

#### Major Issues
<List major issues if any>

#### Positive Highlights
<List 2-3 things done well>

### Full Review
Complete review available at: `agent-feedback/<agent>/<TICKET>.md`

### Next Steps
<What should happen next>

---
*Review by: <agent-name>*
```

**If PR comment fails:**
- Don't fail the review
- Note in feedback file: "Attempted to comment on PR but failed: <reason>"
- Continue and complete feedback file

## Go Code Review Checklist

Use this checklist when analyzing changes:

### 1. Ticket Alignment
- [ ] Changes implement what ticket describes
- [ ] All acceptance criteria addressed
- [ ] No out-of-scope changes
- [ ] Technical approach matches ticket's technical context

### 2. Testing (Critical)
- [ ] Tests use `testify/suite` and `testify/assert`
- [ ] Table-driven tests for validators/parsers
- [ ] Test coverage for new code
- [ ] Tests are not flaky (no time/random dependencies)
- [ ] Tests follow TDD (test file changes before prod code)
- [ ] Edge cases covered

### 3. Go Standards Compliance
- [ ] Follows Effective Go guidelines
- [ ] `gofmt` applied (consistent formatting)
- [ ] No golangci-lint errors
- [ ] Proper error handling (`errors.Is`, `errors.As`)
- [ ] Context as first parameter for I/O functions
- [ ] No global mutable state
- [ ] Concurrency (if any) passes `-race`

### 4. Code Quality
- [ ] Small, focused changes (not a mega-commit)
- [ ] Single responsibility per function/package
- [ ] Clear, descriptive names
- [ ] Comments for non-obvious logic
- [ ] No commented-out code
- [ ] No debug print statements

### 5. Dependencies
- [ ] No unnecessary dependencies added
- [ ] New dependencies are widely adopted
- [ ] Uses stdlib when possible
- [ ] Uses approved defaults (chi, slog, testify)
- [ ] `go.mod` updated correctly

### 6. Architecture
- [ ] No circular dependencies
- [ ] Clear package boundaries
- [ ] Interfaces at consumer side
- [ ] Dependency injection via constructors
- [ ] Domain logic separate from transport

### 7. Error Handling
- [ ] All errors checked
- [ ] Errors wrapped with context
- [ ] No swallowed errors
- [ ] No panic in library code
- [ ] Typed errors at package boundaries

### 8. Security
- [ ] No hardcoded secrets
- [ ] No SQL injection risks
- [ ] No XSS vulnerabilities
- [ ] Input validation present
- [ ] No PII in logs

### 9. Documentation
- [ ] Public APIs documented
- [ ] README updated (if needed)
- [ ] Runbook updated (if infrastructure)
- [ ] Breaking changes noted

### 10. Commits
- [ ] Commit messages follow format: `<TICKET> type(scope): description`
- [ ] Commits are atomic (small, working)
- [ ] No "WIP" or "fix typo" commits

## Example: Complete Review Workflow

**User request:**
> "Review the changes in branch feature/TBY-31-health-check against ticket TBY-31"

**Agent response:**

```
I'll review the changes against TBY-31. Let me:

1. Fetch ticket TBY-31 from Jira
2. Analyze branch changes
3. Check for open PR
4. Write feedback

[Fetches ticket...]

Ticket TBY-31: "Add health check endpoint to API"
Acceptance Criteria:
- GET /health returns 200 OK
- Response includes service status
- Includes basic tests

[Analyzes git diff...]

Changes found:
- pkg/api/handler.go: Added HealthCheck handler
- pkg/api/handler_test.go: Added tests using testify
- docs/api.md: Updated docs

[Checks for PR...]

PR #45 found: https://github.com/org/repo/pull/45

[Performs review...]

Review complete! 

Feedback written to: agent-feedback/claude/TBY-31.md
PR comment added: https://github.com/org/repo/pull/45#issuecomment-123

Summary:
✅ Overall: Approved with minor suggestions
✅ All acceptance criteria met (3/3)
✅ Tests use testify correctly
⚠️ One minor suggestion: Add timeout to health check

Ready to merge after addressing the minor suggestion.
```

## Anti-Patterns: What NOT To Do

### ❌ Reviewing Without Ticket Context

**Bad**:
```
User: "Review branch feature/TBY-31-health-check"
Agent: [Reviews code without fetching TBY-31 from Jira]
       "Code looks good, tests are present"
```

**Correct**:
```
User: "Review branch feature/TBY-31-health-check"
Agent: [Fetches TBY-31 first]
       "Reviewing against TBY-31: 'Add health check endpoint'
        Acceptance criteria: [lists them]
        
        Analysis:
        ✅ Criterion 1 met: GET /health returns 200
        ❌ Criterion 2 NOT met: Response missing service status
        ..."
```

### ❌ Writing Feedback Without Agent Name

**Bad**:
```
# File: agent-feedback/TBY-31.md
# Code Review: TBY-31
...
```

**Correct**:
```
# File: agent-feedback/claude/TBY-31.md
# Code Review: TBY-31

**Reviewer**: claude
...

---
*Review by: claude*
```

### ❌ Not Creating Directory

**Bad**:
```
[Tries to write to agent-feedback/claude/TBY-31.md]
[Fails because directory doesn't exist]
[Gives up]
```

**Correct**:
```bash
# Create directory first
mkdir -p agent-feedback/claude

# Then write file
cat > agent-feedback/claude/TBY-31.md << 'EOF'
...
EOF
```

### ❌ Vague Feedback

**Bad**:
```markdown
## Issues
- Code could be better
- Tests need improvement
- Consider refactoring
```

**Correct**:
```markdown
## Issues

#### Critical
1. **Missing error handling in HealthCheck handler**
   - **File**: `pkg/api/handler.go:45`
   - **Problem**: Database call at line 47 doesn't check error
   - **Fix**: Add `if err != nil` check and return 503 status
   - **Why**: Could panic if database is down

#### Major
2. **Test doesn't verify response body**
   - **File**: `pkg/api/handler_test.go:23`
   - **Problem**: Only checks status code, not response content
   - **Fix**: Add assertion for response body containing "status"
   - **Why**: Acceptance criteria requires service status in response
```

### ❌ Skipping PR Comment Without Noting It

**Bad**:
```
[Cannot access PR to comment]
[Says nothing about it]
[Only creates feedback file]
```

**Correct**:
```
[Cannot access PR to comment]

Feedback file: agent-feedback/claude/TBY-31.md

Note: I cannot access PR to add comment (no GitHub API access).
Full review is in the feedback file. Please share with reviewers.
```

### ❌ Not Verifying Against Acceptance Criteria

**Bad**:
```markdown
## Review Summary
Code looks good. Tests are present. Following Go best practices.
```

**Correct**:
```markdown
## Acceptance Criteria Verification

- [✅] **GET /health returns 200 OK**
  - Evidence: `pkg/api/handler.go:45-48` implements this
  - Test: `pkg/api/handler_test.go:23-30` verifies it

- [❌] **Response includes service status**
  - NOT MET: Response only returns empty body
  - Missing: Need to add status field with "healthy"/"degraded"
  - Fix: Add struct with Status field, marshal to JSON

- [✅] **Includes basic tests**
  - Evidence: `pkg/api/handler_test.go` uses testify suite
  - Coverage: Handler logic covered, edge cases included
```

## Agent-Specific Notes

### For Different Agents

The skill works the same for all agents (codex, copilot, antigravity, warp, claude), but:

**Agent identification:**
- Agent must know its own name
- If unsure, ask user: "Which agent am I?"
- Use lowercase in file paths and signatures

**File path examples:**
```
agent-feedback/codex/TBY-31.md
agent-feedback/copilot/TBY-31.md
agent-feedback/antigravity/TBY-31.md
agent-feedback/warp/TBY-31.md
agent-feedback/claude/TBY-31.md
```

**PR comment signature:**
```markdown
---
*Review by: codex*
```

```markdown
---
*Review by: claude*
```

## Checklist

Before completing review:

- [ ] Fetched ticket from Jira successfully
- [ ] Analyzed git diff and commits
- [ ] Checked against all acceptance criteria
- [ ] Applied Go code review checklist
- [ ] Created `agent-feedback/<agent>/` directory if needed
- [ ] Written feedback to `agent-feedback/<agent>/<TICKET>.md`
- [ ] Checked for open PR
- [ ] Attempted PR comment (if PR exists and possible)
- [ ] Noted PR comment status in feedback file
- [ ] Included agent name in feedback and PR comment
- [ ] Provided clear, actionable feedback
- [ ] Specified next steps (ready to merge, needs fixes, etc.)
