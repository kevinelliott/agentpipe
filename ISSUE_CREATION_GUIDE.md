# Issue Creation Guide

This guide explains how to create GitHub issues from the identified problems in the AgentPipe codebase.

## Quick Start

### Option 1: Automated Creation (Recommended)

If you have GitHub CLI (`gh`) installed and authenticated:

```bash
# Make the script executable (if not already)
chmod +x CREATE_ISSUES.sh

# Run the script to create all issues
./CREATE_ISSUES.sh
```

**Note:** The CREATE_ISSUES.sh script is partially implemented. You may need to complete it with all 12 issues or use Option 2.

### Option 2: Manual Creation

1. Open `GITHUB_ISSUES.md`
2. For each issue, copy the entire section
3. Go to https://github.com/kevinelliott/agentpipe/issues/new
4. Paste the title and description
5. Add the specified labels
6. Click "Submit new issue"

### Option 3: Using GitHub CLI Directly

For each issue in `GITHUB_ISSUES.md`, run:

```bash
gh issue create \
  --title "ISSUE_TITLE" \
  --label "label1,label2,label3" \
  --body "ISSUE_DESCRIPTION"
```

## Files Included

1. **`ISSUES_ANALYSIS.md`** - Comprehensive technical analysis
   - Detailed descriptions of all 12 issues
   - Code locations and examples
   - Impact assessments
   - Recommended fixes
   - Risk evaluations

2. **`GITHUB_ISSUES.md`** - Ready-to-use GitHub issue templates
   - Formatted for direct copy-paste into GitHub
   - All 12 issues with proper markdown
   - Labels already specified
   - Priority ordering

3. **`CREATE_ISSUES.sh`** - Automation script (partial)
   - Shell script to create issues via GitHub CLI
   - May need completion for all 12 issues
   - Requires `gh` CLI and authentication

## Issue Summary

### Total: 12 Issues

#### By Severity:
- **High:** 1 issue
  - Issue 2: API Key Logging Risk (Security)
  
- **Medium:** 6 issues
  - Issue 1: Race Condition in Orchestrator
  - Issue 3: Memory Leak in Rate Limiter Map
  - Issue 5: Unchecked File Operation Errors
  - Issue 7: Event Store File Permissions
  - Issue 8: Config Directory Permissions
  - Issue 10: Unbounded Retry Timeout

- **Low:** 5 issues
  - Issue 4: Integer Overflow in Token Calculation
  - Issue 6: Unseeded Random Number Generator
  - Issue 9: Negative Terminal Width Panic Risk
  - Issue 11: Model Name Validation Missing
  - Issue 12: HTTP Client Timeout Too Long

#### By Category:
- **Security:** 3 issues (#2, #7, #8)
- **Concurrency:** 2 issues (#1, #6)
- **Resource Management:** 2 issues (#3, #10)
- **Error Handling:** 1 issue (#5)
- **Validation:** 1 issue (#11)
- **Defensive Programming:** 1 issue (#9)
- **Arithmetic:** 1 issue (#4)
- **Performance:** 1 issue (#12)

## Recommended Priority Order

1. **Issue 2** - API Key Logging (HIGH - Security) ⚠️
2. **Issue 7** - Event Store File Permissions (MEDIUM - Security)
3. **Issue 8** - Config Directory Permissions (MEDIUM - Security)
4. **Issue 1** - Race Condition (MEDIUM - Concurrency)
5. **Issue 3** - Memory Leak (MEDIUM - Resource Management)
6. **Issue 5** - File Operation Errors (MEDIUM - Error Handling)
7. **Issue 10** - Unbounded Retry (MEDIUM - Resource Exhaustion)
8. **Issues 4, 6, 9, 11, 12** - Low priority enhancements

## Labels to Use

When creating issues, use these label combinations:

### High Priority Issues
- Issue 2: `security`, `high-priority`, `bug`

### Medium Priority Issues
- Issue 1: `bug`, `concurrency`, `medium-priority`
- Issue 3: `bug`, `memory-leak`, `medium-priority`
- Issue 5: `bug`, `error-handling`, `medium-priority`
- Issue 7: `security`, `medium-priority`, `bug`
- Issue 8: `security`, `medium-priority`, `bug`
- Issue 10: `bug`, `resource-exhaustion`, `medium-priority`

### Low Priority Issues
- Issue 4: `bug`, `low-priority`, `enhancement`
- Issue 6: `bug`, `low-priority`, `enhancement`
- Issue 9: `bug`, `low-priority`, `enhancement`
- Issue 11: `enhancement`, `low-priority`, `validation`
- Issue 12: `enhancement`, `low-priority`, `performance`

## Example: Creating Issue 2 (High Priority)

```bash
gh issue create \
  --title "Security: API Key Logging Risk in Bridge Client" \
  --label "security,high-priority,bug" \
  --body "$(cat GITHUB_ISSUES.md | sed -n '/^## Issue 2:/,/^## Issue 3:/p' | head -n -1)"
```

Or manually:
1. Go to https://github.com/kevinelliott/agentpipe/issues/new
2. Copy the content from "Issue 2" in `GITHUB_ISSUES.md`
3. Paste as title and description
4. Add labels: `security`, `high-priority`, `bug`
5. Submit

## Verification

After creating all issues, verify:
- [ ] All 12 issues are created
- [ ] Each has correct labels
- [ ] Descriptions are complete
- [ ] Code examples render correctly
- [ ] Links work (if any)

## Next Steps

After creating issues:
1. Review and prioritize with the team
2. Assign issues to developers
3. Add to project board/milestones
4. Start with high-priority security issues
5. Plan fixes for medium-priority issues
6. Schedule low-priority enhancements

## Additional Resources

- Full analysis: `ISSUES_ANALYSIS.md`
- Issue templates: `GITHUB_ISSUES.md`
- Automation script: `CREATE_ISSUES.sh`

## Support

If you encounter issues creating GitHub issues:
1. Ensure `gh` CLI is installed: `gh --version`
2. Authenticate: `gh auth login`
3. Test: `gh issue list --repo kevinelliott/agentpipe`

For questions about the issues themselves, refer to `ISSUES_ANALYSIS.md` for detailed technical information.
