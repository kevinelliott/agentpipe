# AgentPipe Documentation

Welcome to the AgentPipe documentation directory.

## 📚 Core Documentation

- **[Architecture](architecture.md)** - System architecture and design patterns
- **[Contributing](contributing.md)** - Contribution guidelines
- **[Development](development.md)** - Development setup and workflow
- **[Docker](docker.md)** - Docker deployment guide
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## 🔬 Research Documents

### AI Agent CLI Ecosystem Analysis (October 2025)

A comprehensive research project to identify missing AI agent CLIs that could be added to AgentPipe.

#### Quick Start
👉 **Start here:** [Missing AI Agents Summary](missing-ai-agents-summary.md) - Executive summary with TL;DR

#### Detailed Research
- **[Full Research Report](research-ai-agent-clis.md)** (17KB) - Comprehensive analysis of 20+ AI agent CLIs
  - Current AgentPipe support (15 agents)
  - Missing agents identified
  - Integration complexity assessment
  - Recommendations and priority queue
  
- **[Agent Comparison Matrix](agent-comparison-matrix.md)** (11KB) - Side-by-side comparison table
  - Current vs. missing agents
  - Feature comparison
  - Market position analysis
  - Coverage statistics
  - Decision matrix for evaluating new agents

#### Implementation Guides

Ready-to-use GitHub issue templates for implementing recommended agents:

1. **[Add Aider Support](issue-template-add-aider.md)** ⭐ HIGHEST PRIORITY
   - Git-aware AI coding assistant
   - 18,000+ GitHub stars
   - Integration effort: LOW
   - Implementation time: 1-2 days

2. **[Add Mentat Support](issue-template-add-mentat.md)** ⭐ HIGH PRIORITY
   - Direct file editing AI assistant
   - 2,500+ GitHub stars
   - Integration effort: LOW
   - Implementation time: 1-2 days

3. **[Add GPT Engineer Support](issue-template-add-gpt-engineer.md)** ⭐ MEDIUM-HIGH PRIORITY
   - Whole-project generation tool
   - 52,000+ GitHub stars
   - Integration effort: MEDIUM
   - Implementation time: 2-3 days

### Key Findings

**Answer to "Are we missing any AI agent CLIs?"**
> YES! We're missing several notable ones:

**Top 3 Most Notable Omissions:**
1. **Aider** (18k+ stars) - Most popular open-source AI coding assistant
2. **Mentat** (2.5k+ stars) - AI pair programmer with direct file editing
3. **GPT Engineer** (52k+ stars) - Whole-project generation tool

**Coverage Statistics:**
- **Current:** 15 agents (14 CLI + 1 API)
- **With top 3 additions:** 18 agents (+20% increase)
- **Competitive position:** Would become most comprehensive multi-agent platform

**Recommended Action:**
Implement Aider first (quickest win, highest demand), followed by Mentat and GPT Engineer.

---

## 📖 How to Use These Documents

### For Users
1. Read the [Missing AI Agents Summary](missing-ai-agents-summary.md) to understand what's missing
2. Check the [Agent Comparison Matrix](agent-comparison-matrix.md) for detailed comparisons
3. Vote or comment on GitHub issues for agents you'd like to see added

### For Contributors
1. Read the full [Research Report](research-ai-agent-clis.md) for context
2. Choose an agent to implement from the issue templates
3. Follow the implementation checklist in the respective issue template
4. Refer to existing adapters in `pkg/adapters/` for patterns
5. Submit a PR following [Contributing Guidelines](contributing.md)

### For Maintainers
1. Use issue templates to create GitHub issues
2. Prioritize based on community feedback and research recommendations
3. Track implementation progress against the roadmap
4. Update research documents as new agents emerge

---

## 🚀 Roadmap

Based on research findings, here's the recommended implementation order:

### Phase 1: Quick Wins (Weeks 1-2)
- [ ] **Aider** - Git-aware coding (1-2 days)
- [ ] **Mentat** - File editing (1-2 days)

### Phase 2: Expand Coverage (Weeks 3-4)
- [ ] **GPT Engineer** - Project generation (2-3 days)
- [ ] **Cody** - Sourcegraph AI (2-3 days)

### Phase 3: API Expansion (Weeks 5-6)
- [ ] **Anthropic API** - Direct Claude access (2-3 days)
- [ ] **OpenAI API** - Direct GPT access (2-3 days)
- [ ] **Together AI** - Open-source models (2-3 days)

### Phase 4: Specialized Tools (As Needed)
- [ ] **Sourcery** - Python refactoring
- [ ] **CodeGeeX** - International users

---

## 📊 Research Methodology

The research included:
- Analysis of 20+ AI agent CLI tools
- GitHub star counts and community popularity
- CLI quality assessment (stdin/stdout support, documentation)
- Integration complexity evaluation
- Market positioning analysis
- Feature gap identification

**Sources:**
- GitHub repositories and trending projects
- npm package registry
- PyPI (Python Package Index)
- Developer communities (Reddit, HackerNews, Twitter)
- AI tool review sites and blogs
- Recent product launches (2024-2025)

---

## 🔄 Keeping Research Current

This research was conducted in **October 2025**. The AI agent CLI ecosystem evolves rapidly.

**To keep research current:**
1. Review quarterly for new popular tools
2. Monitor GitHub trending repositories
3. Track npm/PyPI new releases in AI/LLM space
4. Watch developer community discussions
5. Update comparison matrix with new agents
6. Reassess priorities based on community feedback

**Next review scheduled:** January 2026

---

## 📝 Document History

| Date | Document | Action | Notes |
|------|----------|--------|-------|
| 2025-10-29 | All research docs | Created | Initial comprehensive research |
| - | - | - | - |

---

## 🤝 Feedback & Contributions

Have suggestions for agents to add? Found a great new AI CLI tool?

- Open a GitHub issue with the `agent-support` label
- Join the discussion in existing issues
- Submit a PR with updated research
- Share feedback in GitHub Discussions

---

## 📄 License

These research documents are part of the AgentPipe project and are licensed under the same MIT License as the main project.

---

**Happy researching! 🚀**
