# Missing AI Agent CLIs - Executive Summary

**Research Date:** 2025-10-29  
**Current Support:** 14 CLI-based agents + 1 API service  
**Purpose:** Answer "Are we missing any AI agent CLIs that are out there?"

---

## TL;DR - Yes, We're Missing Some Notable Ones! 🎯

**Top 3 Most Notable Omissions:**
1. **Aider** (18k+ ⭐) - Most popular open-source AI coding assistant
2. **Mentat** (2.5k+ ⭐) - AI pair programmer with direct file editing
3. **GPT Engineer** (52k+ ⭐) - Whole-project generation tool

---

## Quick Comparison: What We Have vs What's Missing

### ✅ What AgentPipe DOES Support (14 CLI + 1 API)

**Excellent Coverage:**
- Major commercial tools: Claude, Cursor, Copilot, Factory
- Cloud providers: Gemini, Qwen, Groq
- Open-source tools: Crush, Ollama
- Emerging platforms: Amp, Qoder, OpenCode, Kimi, Codex
- **API access:** OpenRouter (400+ models)

### 🔍 What AgentPipe DOESN'T Support (Top Misses)

**Popular Open-Source Tools:**
- ❌ **Aider** - 18k+ stars, git-aware AI pair programming
- ❌ **Mentat** - 2.5k+ stars, direct file editing
- ❌ **GPT Engineer** - 52k+ stars, whole-project generation
- ❌ **Cody** - 2.5k+ stars, Sourcegraph's AI assistant (sibling to Amp)

**Direct API Integrations:**
- ❌ **Anthropic API** (direct) - Current Claude CLI is a wrapper
- ❌ **OpenAI API** (direct) - Current Codex CLI is a wrapper
- ❌ **Together AI** - Fast inference for open-source models

**Niche/Specialized:**
- ⚠️ **Sourcery** - Python refactoring specialist
- ⚠️ **CodeGeeX** - Chinese/international users focus
- ⚠️ **Continue.dev** - Has LSP mode but primarily VS Code

---

## Recommendation: Should We Add Them?

### 🟢 YES - High Priority (Add These)

#### 1. Aider ⭐⭐⭐ **HIGHEST PRIORITY**
```bash
# Why add:
- 18k+ GitHub stars (extremely popular)
- Excellent CLI interface with stdin support
- Git-aware coding assistant
- Works with GPT-4, Claude, etc.
- Different workflow than existing agents

# How easy:
- Effort: LOW (well-documented CLI)
- Install: pip install aider-chat
- Command: aider --message "prompt"
```

#### 2. Mentat ⭐⭐⭐ **HIGH PRIORITY**
```bash
# Why add:
- 2.5k+ stars, solid adoption
- Direct file editing approach
- CLI-first design
- Alternative to Aider with different UX

# How easy:
- Effort: LOW (simple CLI interface)
- Install: pip install mentat
- Command: mentat (supports stdin)
```

#### 3. GPT Engineer ⭐⭐ **MEDIUM-HIGH PRIORITY**
```bash
# Why add:
- 52k+ stars (hugely popular)
- Different use case: whole-project generation
- Great for bootstrapping new projects
- CLI-first tool

# How easy:
- Effort: MEDIUM (may need adaptation)
- Install: pip install gpt-engineer
- Command: gpte or gpt-engineer
```

### 🟡 MAYBE - Medium Priority (Consider These)

#### 4. Cody (Sourcegraph) ⭐⭐
- From Amp's parent company
- Strong codebase context understanding
- Effort: MEDIUM (requires sg CLI)
- Use case: Complements Amp

#### 5. Anthropic API (Direct) ⭐⭐⭐
- Better than Claude CLI (official API)
- Pattern: Same as OpenRouter
- Effort: LOW
- Impact: HIGH (more direct control)

#### 6. OpenAI API (Direct) ⭐⭐
- Official GPT API access
- Pattern: Same as OpenRouter
- Effort: LOW
- Impact: MEDIUM (already have Codex CLI)

### 🔴 NO - Low Priority or Not Suitable

- **Tabnine**: IDE plugin focus, poor CLI
- **Continue.dev**: Better as VS Code extension
- **Tabby**: Server-based, not CLI-friendly
- **Sweep**: GitHub App, not conversational
- **Devin**: Closed beta, no API/CLI
- **Replit Agent**: Web-based only
- **bolt.new**: Web-based only

---

## Impact Analysis

### If We Add Top 3 (Aider, Mentat, GPT Engineer)

**Coverage Improvement:**
- Current: 14 CLI + 1 API = 15 total agents
- With additions: 17 CLI + 1 API = 18 total agents
- **+20% increase in agent options**

**User Segments Unlocked:**
- **Git-aware coding:** Aider users (large community)
- **File-editing focus:** Mentat users (alternative workflow)
- **Project generation:** GPT Engineer users (bootstrapping)

**Competitive Position:**
- Would have **most comprehensive multi-agent orchestration**
- Only platform supporting Aider + Mentat + GPT Engineer together
- Clear differentiation from competitors

### If We Add Direct APIs (Anthropic, OpenAI)

**Benefits:**
- Better rate limits than CLI wrappers
- More features (function calling, etc.)
- Lower latency
- Official support
- Follows OpenRouter pattern (proven)

**Effort:**
- LOW (reuse OpenRouter implementation pattern)
- Each API: ~1-2 days implementation + testing

---

## Implementation Roadmap

### Phase 1: Quick Wins (Week 1-2)
**Goal:** Add most-requested CLI agents
1. **Aider** - Day 1-2
   - Create `pkg/adapters/aider.go`
   - Test with `--message` flag
   - Add to agent registry
   
2. **Mentat** - Day 3-4
   - Create `pkg/adapters/mentat.go`
   - Test stdin interface
   - Add to agent registry

### Phase 2: Expand Coverage (Week 3-4)
**Goal:** Add differentiated tools
3. **GPT Engineer** - Day 5-7
   - Create `pkg/adapters/gpt_engineer.go`
   - Handle project generation workflow
   - Test multi-file generation
   
4. **Cody** - Day 8-10
   - Create `pkg/adapters/cody.go`
   - Integrate with sg CLI
   - Test codebase context features

### Phase 3: API Expansion (Week 5-6)
**Goal:** Direct provider APIs
5. **Anthropic API** - Day 11-13
   - Create `pkg/adapters/anthropic_api.go`
   - Reuse OpenRouter HTTP client pattern
   - Add streaming support
   
6. **OpenAI API** - Day 14-16
   - Create `pkg/adapters/openai_api.go`
   - OpenAI-compatible client
   - Function calling support

### Phase 4: Polish (Week 7)
**Goal:** Documentation and testing
- Update README with new agents
- Add example configurations
- Integration tests
- Update doctor command
- Release notes

---

## User Communication

### For GitHub Issue Response:

**Short Answer:**
> Yes! We're missing some popular ones, especially **Aider** (18k+ stars), **Mentat**, and **GPT Engineer** (52k+ stars). We have excellent coverage of commercial tools but could expand open-source support.

**Detailed Answer:**
> After comprehensive research, we identified several notable omissions:
> 
> **High Priority Additions:**
> 1. Aider - Most popular open-source AI coding assistant (18k+ stars)
> 2. Mentat - AI pair programmer with direct file editing (2.5k+ stars)
> 3. GPT Engineer - Whole-project generation (52k+ stars)
> 4. Cody - Sourcegraph's AI assistant (sibling to Amp)
>
> **API-Based Additions:**
> 5. Anthropic API (direct) - Better than Claude CLI wrapper
> 6. OpenAI API (direct) - Direct GPT access
>
> See full research document in `docs/research-ai-agent-clis.md` for complete analysis.

### For README Update:

Add a "Roadmap" or "Coming Soon" section:

```markdown
## Coming Soon

We're actively working on expanding agent support. Upcoming additions:

### Planned CLI Agents
- 🔄 **Aider** - Git-aware AI pair programming
- 🔄 **Mentat** - Direct file editing AI assistant  
- 🔄 **GPT Engineer** - Whole-project generation
- 🔄 **Cody** - Sourcegraph's AI assistant

### Planned API Integrations
- 🔄 **Anthropic API** - Direct Claude API access
- 🔄 **OpenAI API** - Direct GPT API access

Want to see another agent supported? [Open an issue](https://github.com/kevinelliott/agentpipe/issues)!
```

---

## Testing Strategy

For each new agent, ensure:

1. **Doctor Command Detection:**
   - Add to supported agents list
   - Detection via `which <command>`
   - Version checking
   - Installation instructions

2. **Basic Functionality:**
   - Initialize with config
   - Health check passes
   - Send single message
   - Receive response

3. **Multi-Agent Conversation:**
   - Participate in round-robin mode
   - Respond in reactive mode
   - Handle message filtering
   - Respect turn timeouts

4. **Edge Cases:**
   - Empty responses
   - Error handling
   - Timeout scenarios
   - Long responses

5. **Documentation:**
   - README entry
   - Example configuration
   - Troubleshooting section
   - Installation guide

---

## Conclusion

**Are we missing AI agent CLIs?**
✅ **YES** - We're missing some very popular ones!

**Should we add them?**
✅ **YES** - Especially Aider, Mentat, and GPT Engineer

**How much work?**
✅ **LOW-MEDIUM** - All have good CLI interfaces

**Impact?**
✅ **HIGH** - Would make AgentPipe the most comprehensive orchestration platform

**Recommendation:**
Start with **Aider** (quickest win, highest demand), then **Mentat**, then **GPT Engineer**. This would significantly strengthen AgentPipe's position in the market.

---

## Next Steps

1. ✅ Research completed - see `docs/research-ai-agent-clis.md`
2. ⬜ Prioritize based on user feedback
3. ⬜ Create implementation issues for top 3
4. ⬜ Begin Phase 1 implementation
5. ⬜ Update roadmap in README

**Questions?** Review the full research document or open a GitHub discussion!
