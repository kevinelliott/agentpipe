# Research: AI Agent CLI Ecosystem Analysis

**Date:** 2025-10-29  
**Purpose:** Identify AI agent CLIs not currently supported by AgentPipe  
**Current Support:** 14 CLI-based agents + 1 API-based agent (OpenRouter)

---

## Currently Supported Agents

AgentPipe currently supports the following AI agent CLIs:

### CLI-Based Agents (14)
1. **Amp** (Sourcegraph) - `amp` - Advanced coding agent with autonomous reasoning
2. **Claude** (Anthropic) - `claude` - Advanced reasoning and coding  
3. **Codex** (OpenAI) - `codex` - Code generation specialist
4. **Copilot** (GitHub) - `github-copilot-cli` - Terminal-based coding agent
5. **Crush** (Charm/Charmbracelet) - `crush` - Terminal-first AI coding assistant
6. **Cursor** (Cursor AI) - `cursor-agent` - IDE-integrated AI assistance
7. **Factory** (Factory.ai) - `droid` - Agent-native software development
8. **Gemini** (Google) - `gemini` - Multimodal understanding
9. **Groq** - `groq` - Fast AI code assistant powered by Groq LPUs
10. **Kimi** (Moonshot AI) - `kimi` - Interactive AI agent with advanced reasoning
11. **OpenCode** (SST) - `opencode` - AI coding agent built for the terminal
12. **Qoder** - `qodercli` - Agentic coding platform with enhanced context engineering
13. **Qwen** (Alibaba) - `qwen` - Multilingual capabilities
14. **Ollama** - `ollama` - Local LLM support (planned)

### API-Based Agents (1)
15. **OpenRouter** - API access to 400+ models from multiple providers

---

## Research Methodology

To identify missing AI agent CLIs, I researched:
1. GitHub repositories with AI agent/coding assistant tags
2. npm packages in the AI/LLM space
3. Popular AI development tools and frameworks
4. Developer communities and social media discussions
5. Recent AI product launches (2024-2025)

---

## Potentially Missing AI Agent CLIs

### 1. **Aider** ⭐ HIGH PRIORITY
- **Command:** `aider`
- **Provider:** Paul Gauthier (open source)
- **GitHub:** https://github.com/paul-gauthier/aider
- **Install:** `pip install aider-chat` or `pipx install aider-chat`
- **Description:** AI pair programming in your terminal with git integration
- **Stars:** ~18k+ GitHub stars (very popular)
- **Features:**
  - Git-aware coding assistant
  - Works with GPT-4, Claude, etc.
  - Auto-commits changes
  - Multi-file editing
  - Works via stdin/CLI
- **Why Add:** One of the most popular open-source AI coding assistants
- **CLI Support:** ✅ Excellent - has `--message` flag for non-interactive use
- **Integration Complexity:** LOW - stdin/stdout based

### 2. **Continue.dev** (CLI/LSP mode) ⭐ MEDIUM PRIORITY
- **Command:** `continue`
- **Provider:** Continue.dev (open source)
- **GitHub:** https://github.com/continuedev/continue
- **Install:** VS Code extension (has CLI/LSP server mode)
- **Description:** Open-source AI code assistant with LSP support
- **Stars:** ~16k+ GitHub stars
- **Features:**
  - Works with multiple LLM providers
  - Context-aware code suggestions
  - LSP server mode for CLI integration
- **Why Add:** Popular VS Code extension with CLI capabilities
- **CLI Support:** ⚠️ MODERATE - primarily VS Code, but has LSP mode
- **Integration Complexity:** MEDIUM - requires LSP integration

### 3. **Tabnine** (CLI mode)
- **Command:** `tabnine-cli` or `tabnine`
- **Provider:** Tabnine (commercial)
- **Website:** https://www.tabnine.com/
- **Install:** Via their installer
- **Description:** AI code completion with privacy focus
- **Features:**
  - On-premise/cloud options
  - Multi-language support
  - Team learning
- **Why Add:** Enterprise-focused with privacy features
- **CLI Support:** ⚠️ LIMITED - primarily IDE plugin
- **Integration Complexity:** HIGH - not designed for CLI use

### 4. **Cody** (Sourcegraph) ⭐ MEDIUM-HIGH PRIORITY
- **Command:** `cody` or `sg cody`
- **Provider:** Sourcegraph (Amp's parent company)
- **GitHub:** https://github.com/sourcegraph/cody
- **Install:** Part of Sourcegraph CLI
- **Description:** AI coding assistant with codebase context
- **Stars:** ~2.5k+ GitHub stars
- **Features:**
  - Uses codebase context
  - Multi-LLM support
  - Code search integration
- **Why Add:** From Sourcegraph (same as Amp), strong codebase understanding
- **CLI Support:** ✅ Good - has CLI mode via `sg cody chat`
- **Integration Complexity:** LOW-MEDIUM - similar to Amp

### 5. **GPT Engineer** ⭐ MEDIUM PRIORITY
- **Command:** `gpt-engineer` or `gpte`
- **Provider:** Anton Osika (open source)
- **GitHub:** https://github.com/gpt-engineer-org/gpt-engineer
- **Install:** `pip install gpt-engineer`
- **Description:** Specify what you want to build, AI creates entire codebase
- **Stars:** ~52k+ GitHub stars (very popular)
- **Features:**
  - Whole-app generation
  - Iterative development
  - Git integration
- **Why Add:** Extremely popular for bootstrapping projects
- **CLI Support:** ✅ Excellent - CLI-first design
- **Integration Complexity:** LOW - simple CLI interface

### 6. **Tabby** (CLI mode)
- **Command:** `tabby`
- **Provider:** TabbyML (open source)
- **GitHub:** https://github.com/TabbyML/tabby
- **Install:** Binary download or `cargo install tabby`
- **Description:** Self-hosted AI coding assistant
- **Stars:** ~22k+ GitHub stars
- **Features:**
  - Self-hosted
  - Privacy-focused
  - RAG support
  - Multiple model support
- **Why Add:** Popular self-hosted option
- **CLI Support:** ⚠️ LIMITED - primarily server mode
- **Integration Complexity:** HIGH - server-based architecture

### 7. **Sourcery** (CLI mode)
- **Command:** `sourcery`
- **Provider:** Sourcery AI (commercial)
- **Website:** https://sourcery.ai/
- **Install:** `pip install sourcery-cli`
- **Description:** Python code review and refactoring
- **Features:**
  - Code quality improvements
  - Automated refactoring
  - Python-focused
- **Why Add:** Strong Python focus
- **CLI Support:** ✅ Good - has CLI mode
- **Integration Complexity:** MEDIUM - Python-specific

### 8. **Sweep** (CLI mode)
- **Command:** `sweep`
- **Provider:** Sweep AI (open source)
- **GitHub:** https://github.com/sweepai/sweep
- **Install:** GitHub App + CLI
- **Description:** AI junior developer for your repository
- **Stars:** ~7k+ GitHub stars
- **Features:**
  - Issue-to-PR automation
  - Code understanding
  - GitHub integration
- **Why Add:** Interesting workflow automation
- **CLI Support:** ⚠️ LIMITED - primarily GitHub App
- **Integration Complexity:** HIGH - GitHub-centric

### 9. **AutoGPT** / **GPT-4 Terminal**
- **Command:** `autogpt` or `gpt4-cli`
- **Provider:** Various implementations
- **Description:** Autonomous GPT-4 agents
- **Why Skip:** Too autonomous, not conversational
- **CLI Support:** Varies
- **Integration Complexity:** HIGH

### 10. **Mentat** ⭐ HIGH PRIORITY
- **Command:** `mentat`
- **Provider:** AbanteAI (open source)
- **GitHub:** https://github.com/AbanteAI/mentat
- **Install:** `pip install mentat`
- **Description:** AI coding assistant that writes code directly to files
- **Stars:** ~2.5k+ GitHub stars
- **Features:**
  - Direct file editing
  - Git integration
  - Multi-file changes
  - Works with various LLM APIs
- **Why Add:** Similar to Aider, but different approach
- **CLI Support:** ✅ Excellent - CLI-first with conversation mode
- **Integration Complexity:** LOW - simple CLI interface

### 11. **CodeWhisperer CLI** (AWS)
- **Command:** `codewhisperer` or via AWS CLI
- **Provider:** Amazon (AWS)
- **Website:** https://aws.amazon.com/codewhisperer/
- **Description:** AWS's AI code companion
- **Why Skip:** Primarily IDE plugin, limited CLI
- **CLI Support:** ⚠️ VERY LIMITED
- **Integration Complexity:** HIGH

### 12. **Windsurf** (Codeium)
- **Command:** `windsurf` (if CLI exists)
- **Provider:** Codeium
- **Website:** https://codeium.com/windsurf
- **Description:** Agentic IDE (recently launched)
- **Why Research:** New agentic IDE from Codeium
- **CLI Support:** ❓ UNKNOWN - very new (late 2024)
- **Integration Complexity:** UNKNOWN

### 13. **Replit Agent** / **Replit AI**
- **Command:** N/A (web-based)
- **Provider:** Replit
- **Why Skip:** Web-based, no CLI
- **CLI Support:** ❌ None
- **Integration Complexity:** N/A

### 14. **Devin** (Cognition Labs)
- **Command:** N/A (web-based)
- **Provider:** Cognition Labs
- **Why Skip:** Closed beta, web-based, no CLI
- **CLI Support:** ❌ None
- **Integration Complexity:** N/A

### 15. **BoltAI** / **bolt.new**
- **Command:** N/A (web-based)
- **Provider:** StackBlitz
- **Why Skip:** Web-based IDE
- **CLI Support:** ❌ None
- **Integration Complexity:** N/A

### 16. **CodeGeeX** ⭐ LOW-MEDIUM PRIORITY
- **Command:** `codegeex`
- **Provider:** Tsinghua University / Zhipu AI (open source)
- **GitHub:** https://github.com/THUDM/CodeGeeX
- **Install:** IDE plugin or API
- **Description:** Multilingual code generation model
- **Stars:** ~8k+ GitHub stars
- **Features:**
  - Multilingual (especially Chinese)
  - Open-source model
  - VS Code plugin
- **Why Add:** Strong international user base
- **CLI Support:** ⚠️ LIMITED - primarily IDE/API
- **Integration Complexity:** MEDIUM-HIGH

### 17. **WizardCoder** / **WizardLM**
- **Command:** N/A (model weights only)
- **Provider:** Microsoft/WizardLM team
- **Why Skip:** Model only, no CLI interface
- **CLI Support:** ❌ None
- **Integration Complexity:** N/A

### 18. **StarCoder** / **StarChat**
- **Command:** N/A (model weights only)
- **Provider:** Hugging Face / BigCode
- **Why Skip:** Model only, no official CLI
- **CLI Support:** ❌ None
- **Integration Complexity:** N/A

### 19. **Phind** (via API)
- **Command:** N/A (web-based search engine)
- **Provider:** Phind
- **Why Skip:** Web-based, developer search engine
- **CLI Support:** ❌ Limited to API
- **Integration Complexity:** MEDIUM (API-based)

### 20. **Perplexity CLI** (if exists)
- **Command:** `perplexity` (unofficial)
- **Provider:** Perplexity AI
- **Why Research:** Popular AI search, may have CLI wrappers
- **CLI Support:** ⚠️ Third-party only
- **Integration Complexity:** MEDIUM

---

## API-Based Services (Alternative to CLIs)

Several services don't have CLIs but offer APIs that could be integrated like OpenRouter:

### High Priority API Services

1. **Anthropic API** (Direct)
   - Why: Official Claude API (vs CLI wrapper)
   - Integration: Similar to OpenRouter pattern
   - Benefits: Direct access, better rate limits, newer models

2. **OpenAI API** (Direct)
   - Why: Official GPT API
   - Integration: Similar to OpenRouter pattern
   - Benefits: Direct access, function calling, newer models

3. **Google AI Studio / Gemini API** (Direct)
   - Why: Official Gemini API
   - Integration: Similar to OpenRouter pattern
   - Benefits: Direct access, multimodal support

4. **Cohere** (API)
   - Why: Command-R+ model, good for enterprise
   - Integration: REST API
   - Benefits: Specialized models for RAG

5. **Together AI** (API)
   - Why: Fast inference, many open-source models
   - Integration: OpenAI-compatible API
   - Benefits: Similar to OpenRouter but different model selection

6. **Replicate** (API)
   - Why: Run any open-source model
   - Integration: REST API
   - Benefits: Widest model selection

---

## Recommendations

### Tier 1: High Priority Additions (Should Add)

1. **Aider** ⭐⭐⭐
   - **Rationale:** 18k+ stars, extremely popular, excellent CLI interface
   - **Effort:** LOW - well-documented CLI with stdin/stdout
   - **Impact:** HIGH - fills gap for git-aware coding assistant
   - **Install:** `pip install aider-chat`
   - **Command:** `aider --message "prompt"`

2. **Mentat** ⭐⭐⭐
   - **Rationale:** Similar to Aider but different workflow, 2.5k+ stars
   - **Effort:** LOW - CLI-first design
   - **Impact:** MEDIUM-HIGH - alternative to Aider with file-editing focus
   - **Install:** `pip install mentat`
   - **Command:** `mentat` (interactive, stdin support)

3. **GPT Engineer** ⭐⭐
   - **Rationale:** 52k+ stars, popular for project bootstrapping
   - **Effort:** MEDIUM - CLI interface, may need adaptation
   - **Impact:** MEDIUM - different use case (whole-project generation)
   - **Install:** `pip install gpt-engineer`
   - **Command:** `gpte` or `gpt-engineer`

4. **Cody** (Sourcegraph) ⭐⭐
   - **Rationale:** From Amp's parent company, strong codebase context
   - **Effort:** MEDIUM - requires Sourcegraph CLI setup
   - **Impact:** MEDIUM - complements Amp with different features
   - **Install:** Part of `sg` CLI
   - **Command:** `sg cody chat`

### Tier 2: Medium Priority (Consider Adding)

5. **Sourcery** (Python focus) ⭐
   - **Rationale:** Strong Python refactoring capabilities
   - **Effort:** MEDIUM - Python-specific
   - **Impact:** LOW-MEDIUM - niche but useful
   - **Install:** `pip install sourcery-cli`

6. **CodeGeeX** (International) ⭐
   - **Rationale:** Good for Chinese/international users
   - **Effort:** MEDIUM-HIGH - API/plugin based
   - **Impact:** LOW-MEDIUM - fills international gap

### Tier 3: API-Based Priority (OpenRouter Pattern)

7. **Anthropic API** (Direct) ⭐⭐⭐
   - **Rationale:** Official API, better than CLI wrapper
   - **Effort:** LOW - similar to OpenRouter implementation
   - **Impact:** HIGH - direct access to Claude models

8. **OpenAI API** (Direct) ⭐⭐
   - **Rationale:** Official API for GPT models
   - **Effort:** LOW - OpenAI-compatible
   - **Impact:** MEDIUM - already have Codex CLI

9. **Together AI** ⭐
   - **Rationale:** Fast inference, open-source models
   - **Effort:** LOW - OpenAI-compatible API
   - **Impact:** MEDIUM - similar to OpenRouter

### Should NOT Add (and Why)

- **Tabnine:** Primarily IDE plugin, poor CLI support
- **Continue.dev:** Better as VS Code extension, complex LSP integration
- **Tabby:** Server-based architecture, not CLI-friendly
- **Sweep:** GitHub App focus, not conversational
- **Devin:** Closed beta, no API/CLI
- **Replit Agent:** Web-based only
- **bolt.new:** Web-based only
- **AutoGPT:** Too autonomous for multi-agent conversations

---

## Implementation Priority Queue

### Phase 1: Quick Wins (CLI-based, low effort)
1. **Aider** - Most popular, best CLI interface
2. **Mentat** - Similar to Aider, easy integration

### Phase 2: Expand Coverage (Medium effort)
3. **GPT Engineer** - Different use case (whole-project generation)
4. **Cody** - Sourcegraph ecosystem, good codebase context

### Phase 3: API Expansion (OpenRouter pattern)
5. **Anthropic API** - Direct Claude API access
6. **OpenAI API** - Direct GPT API access
7. **Together AI** - Open-source model access

### Phase 4: Specialized Tools (If demand exists)
8. **Sourcery** - Python-specific refactoring
9. **CodeGeeX** - International users

---

## Technical Considerations

### CLI Integration Requirements
For a CLI to be easily integrated into AgentPipe:

✅ **Required:**
- Non-interactive mode support (stdin or `--message` flag)
- Text-based input/output
- Reasonable response times (< 60 seconds typical)
- Error handling and status codes

✅ **Nice to Have:**
- Streaming output support
- JSON output mode
- Model selection options
- Temperature/parameter controls
- Version checking

❌ **Deal Breakers:**
- Web-only interface
- No CLI interface
- Requires GUI interaction
- Server-only mode without CLI client
- Closed beta / no public access

### API Integration Requirements (OpenRouter Pattern)
For API-based services:

✅ **Required:**
- REST API with clear documentation
- Authentication via API key
- Streaming support (SSE or similar)
- Usage/token reporting
- Clear pricing information

✅ **Nice to Have:**
- OpenAI-compatible API format
- Retry handling
- Rate limit headers
- Multiple model options

---

## Community Input

This research should be validated against:
1. **GitHub issues/discussions** - What are users asking for?
2. **Social media mentions** - What agents are trending?
3. **Developer surveys** - Stack Overflow, State of Dev tools
4. **AgentPipe usage analytics** - Which agents are most used?

---

## Conclusion

### Summary of Findings

**Current State:** AgentPipe supports 14 CLI-based agents + 1 API service (OpenRouter)

**Key Gaps Identified:**
1. **Aider** - Most glaring omission (18k+ stars, very popular)
2. **Mentat** - Similar to Aider, different approach
3. **GPT Engineer** - Popular for project generation (52k+ stars)
4. **Direct API integrations** - Anthropic/OpenAI APIs (vs CLI wrappers)

**Recommended Actions:**
1. **Immediate:** Add Aider support (HIGH demand, LOW effort)
2. **Short-term:** Add Mentat and GPT Engineer
3. **Medium-term:** Add direct Anthropic/OpenAI API support
4. **Long-term:** Consider Cody, Sourcery based on demand

**Overall Assessment:** AgentPipe has excellent coverage of major AI CLIs. The main gaps are:
- Popular open-source tools (Aider, Mentat, GPT Engineer)
- Direct API integrations for major providers (Anthropic, OpenAI)
- Some niche/specialized tools (Sourcery for Python, CodeGeeX for international)

The additions recommended above would make AgentPipe the most comprehensive multi-agent orchestration platform available.

---

## References

- Aider: https://github.com/paul-gauthier/aider
- Mentat: https://github.com/AbanteAI/mentat
- GPT Engineer: https://github.com/gpt-engineer-org/gpt-engineer
- Cody: https://github.com/sourcegraph/cody
- Continue: https://github.com/continuedev/continue
- Tabby: https://github.com/TabbyML/tabby
- CodeGeeX: https://github.com/THUDM/CodeGeeX
- Sweep: https://github.com/sweepai/sweep

