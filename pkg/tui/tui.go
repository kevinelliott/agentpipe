package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	agentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	systemStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("244"))

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	searchStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	searchMatchStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("226"))
)

type Model struct {
	ctx                context.Context
	config             *config.Config
	agents             []agent.Agent
	messages           []agent.Message
	viewport           viewport.Model
	textarea           textarea.Model
	searchInput        textinput.Model
	commandInput       textinput.Model
	searchMode         bool
	commandMode        bool
	showHelp           bool
	searchResults      []int // Message indices that match search
	currentSearchIndex int   // Current position in searchResults
	filterAgent        string // Agent name to filter by (empty = no filter)
	width              int
	height             int
	ready              bool
	running            bool
	err                error
	statusMessage      string // Temporary status message
}

type messageUpdate struct {
	message agent.Message
}

type conversationDone struct{}

type errMsg struct {
	err error
}

func Run(ctx context.Context, cfg *config.Config, agents []agent.Agent) error {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search messages..."
	searchInput.CharLimit = 100

	commandInput := textinput.New()
	commandInput.Placeholder = "Enter command (filter <agent> | clear)..."
	commandInput.CharLimit = 100

	m := Model{
		ctx:                ctx,
		config:             cfg,
		agents:             agents,
		messages:           make([]agent.Message, 0),
		running:            false,
		searchInput:        searchInput,
		commandInput:       commandInput,
		searchMode:         false,
		commandMode:        false,
		searchResults:      make([]int, 0),
		currentSearchIndex: -1,
		filterAgent:        "",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.startConversation(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle command mode keys
		if m.commandMode {
			switch msg.Type {
			case tea.KeyEsc:
				// Exit command mode
				m.commandMode = false
				m.commandInput.SetValue("")
				return m, nil
			case tea.KeyEnter:
				// Execute command
				m.executeCommand()
				m.commandMode = false
				m.commandInput.SetValue("")
				return m, nil
			default:
				// Update command input
				var cmd tea.Cmd
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		}

		// Handle search mode keys
		if m.searchMode {
			switch msg.Type {
			case tea.KeyEsc:
				// Exit search mode
				m.searchMode = false
				m.searchInput.SetValue("")
				m.searchResults = make([]int, 0)
				m.currentSearchIndex = -1
				return m, nil
			case tea.KeyEnter:
				// Perform search
				m.performSearch()
				return m, nil
			default:
				// Handle other keys in search input
				switch msg.String() {
				case "n":
					// Next search result
					if len(m.searchResults) > 0 {
						m.currentSearchIndex = (m.currentSearchIndex + 1) % len(m.searchResults)
						m.scrollToSearchResult()
					}
					return m, nil
				case "N":
					// Previous search result
					if len(m.searchResults) > 0 {
						m.currentSearchIndex--
						if m.currentSearchIndex < 0 {
							m.currentSearchIndex = len(m.searchResults) - 1
						}
						m.scrollToSearchResult()
					}
					return m, nil
				default:
					// Update search input
					var cmd tea.Cmd
					m.searchInput, cmd = m.searchInput.Update(msg)
					return m, cmd
				}
			}
		}

		// Handle normal mode keys
		switch msg.String() {
		case "/":
			// Enter command mode
			if m.ready && !m.searchMode && !m.showHelp {
				m.commandMode = true
				return m, nil
			}
		case "?":
			// Toggle help modal
			if m.ready && !m.searchMode && !m.commandMode {
				m.showHelp = !m.showHelp
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			// Close help modal if open, otherwise quit
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyCtrlF:
			// Enter search mode (only if ready)
			if m.ready {
				m.searchMode = true
				// Don't call Focus() to avoid cursor initialization issues in tests
				// The searchMode flag will route events to searchInput
				return m, nil
			}
		case tea.KeyCtrlS:
			if !m.running {
				m.running = true
				cmds = append(cmds, m.startConversation())
			}
		case tea.KeyCtrlP:
			m.running = !m.running
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-7)
			m.viewport.SetContent(m.renderMessages())

			ta := textarea.New()
			ta.Placeholder = "Type a message to inject into the conversation..."
			ta.ShowLineNumbers = false
			ta.SetWidth(msg.Width - 4)
			ta.SetHeight(3)
			m.textarea = ta

			// Initialize search input
			searchInput := textinput.New()
			searchInput.Placeholder = "Search messages..."
			searchInput.CharLimit = 100
			// Initialize the internal cursor by updating with a dummy message
			searchInput, _ = searchInput.Update(nil)
			m.searchInput = searchInput

			// Initialize command input
			commandInput := textinput.New()
			commandInput.Placeholder = "Enter command (filter <agent> | clear)..."
			commandInput.CharLimit = 100
			commandInput, _ = commandInput.Update(nil)
			m.commandInput = commandInput

			// Initialize search state if not already set
			if m.searchResults == nil {
				m.searchResults = make([]int, 0)
			}
			if m.currentSearchIndex == 0 {
				m.currentSearchIndex = -1
			}

			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 7
		}

	case messageUpdate:
		m.messages = append(m.messages, msg.message)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case conversationDone:
		m.running = false

	case errMsg:
		m.err = msg.err
		m.running = false
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Show help modal if active
	if m.showHelp {
		return m.renderHelp()
	}

	var b strings.Builder

	title := titleStyle.Render("🚀 AgentPipe - Multi-Agent Conversation")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	status := fmt.Sprintf("Agents: %d | Mode: %s | ", len(m.agents), m.config.Orchestrator.Mode)
	if m.running {
		status += "Status: 🟢 Running"
	} else {
		status += "Status: 🔴 Stopped"
	}
	b.WriteString(statusStyle.Render(status))
	b.WriteString("\n")

	help := helpStyle.Render("?: Help | Ctrl+C: Quit | Ctrl+S: Start | Ctrl+P: Pause/Resume | Ctrl+F: Search | /: Command | ↑↓: Scroll")
	b.WriteString(help)

	// Show filter status
	if m.filterAgent != "" {
		b.WriteString("\n")
		filterStatus := searchStyle.Render(fmt.Sprintf("Filter: %s", m.filterAgent))
		b.WriteString(filterStatus)
	}

	// Show status message if present
	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render(m.statusMessage))
	}

	// Show command bar when in command mode
	if m.commandMode {
		b.WriteString("\n")
		commandBar := searchStyle.Render("/") + m.commandInput.View()
		b.WriteString(commandBar)
	}

	// Show search bar when in search mode
	if m.searchMode {
		b.WriteString("\n")
		searchBar := searchStyle.Render("Search: ") + m.searchInput.View()
		if len(m.searchResults) > 0 {
			searchBar += fmt.Sprintf(" (%d/%d matches, n/N to navigate)", m.currentSearchIndex+1, len(m.searchResults))
		} else if m.searchInput.Value() != "" {
			searchBar += " (no matches)"
		}
		b.WriteString(searchBar)
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return b.String()
}

func (m Model) renderMessages() string {
	var b strings.Builder

	for _, msg := range m.messages {
		// Apply filter if active
		if m.filterAgent != "" && msg.AgentName != m.filterAgent && msg.Role != "system" {
			continue
		}

		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")

		var prefix string
		var style lipgloss.Style

		if msg.Role == "system" {
			prefix = fmt.Sprintf("[%s] System", timestamp)
			style = systemStyle
		} else {
			prefix = fmt.Sprintf("[%s] %s", timestamp, msg.AgentName)
			style = agentStyle
		}

		b.WriteString(style.Render(prefix))
		b.WriteString("\n")
		b.WriteString(messageStyle.Render(msg.Content))
		b.WriteString("\n\n")
	}

	return b.String()
}

// executeCommand parses and executes slash commands
func (m *Model) executeCommand() {
	command := strings.TrimSpace(m.commandInput.Value())
	if command == "" {
		return
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "filter":
		if len(parts) < 2 {
			m.statusMessage = "Usage: filter <agent-name>"
			return
		}
		agentName := parts[1]

		// Check if agent exists
		agentExists := false
		for _, agent := range m.agents {
			if agent.GetName() == agentName {
				agentExists = true
				break
			}
		}

		if !agentExists {
			m.statusMessage = fmt.Sprintf("Agent '%s' not found", agentName)
			return
		}

		m.filterAgent = agentName
		m.statusMessage = fmt.Sprintf("Filtering by agent: %s", agentName)

		// Update viewport with filtered messages
		m.viewport.SetContent(m.renderMessages())

	case "clear":
		if m.filterAgent == "" {
			m.statusMessage = "No filter active"
		} else {
			m.filterAgent = ""
			m.statusMessage = "Filter cleared"

			// Update viewport to show all messages
			m.viewport.SetContent(m.renderMessages())
		}

	default:
		m.statusMessage = fmt.Sprintf("Unknown command: %s", parts[0])
	}
}

// renderHelp displays the help modal with all keybindings
func (m Model) renderHelp() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("📖 AgentPipe - Keyboard Shortcuts Help")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Create help sections
	helpSections := []struct {
		title string
		items []struct {
			key  string
			desc string
		}
	}{
		{
			title: "General Controls",
			items: []struct {
				key  string
				desc string
			}{
				{"Ctrl+C", "Quit application"},
				{"Esc", "Quit application (or close modal)"},
				{"?", "Toggle this help screen"},
				{"↑↓", "Scroll through conversation"},
			},
		},
		{
			title: "Conversation Controls",
			items: []struct {
				key  string
				desc string
			}{
				{"Ctrl+S", "Start conversation"},
				{"Ctrl+P", "Pause/Resume conversation"},
			},
		},
		{
			title: "Search",
			items: []struct {
				key  string
				desc string
			}{
				{"Ctrl+F", "Enter search mode"},
				{"Enter", "Perform search (in search mode)"},
				{"n", "Next search result"},
				{"N", "Previous search result"},
				{"Esc", "Exit search mode"},
			},
		},
		{
			title: "Commands (Slash Commands)",
			items: []struct {
				key  string
				desc string
			}{
				{"/", "Enter command mode"},
				{"filter <agent>", "Filter messages by agent name"},
				{"clear", "Clear active filter"},
				{"Esc", "Exit command mode"},
			},
		},
	}

	// Render help sections
	for _, section := range helpSections {
		sectionTitle := agentStyle.Render(section.title + ":")
		b.WriteString(sectionTitle)
		b.WriteString("\n")

		for _, item := range section.items {
			keyStyle := searchStyle.Render(fmt.Sprintf("  %-15s", item.key))
			b.WriteString(keyStyle)
			b.WriteString("  ")
			b.WriteString(item.desc)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Footer
	footer := helpStyle.Render("Press ? or Esc to close this help screen")
	b.WriteString("\n")
	b.WriteString(footer)

	return b.String()
}

// performSearch searches through messages for the search term
func (m *Model) performSearch() {
	searchTerm := strings.ToLower(m.searchInput.Value())
	if searchTerm == "" {
		m.searchResults = make([]int, 0)
		m.currentSearchIndex = -1
		return
	}

	// Clear previous results
	m.searchResults = make([]int, 0)

	// Search through all messages
	for i, msg := range m.messages {
		// Search in message content and agent name
		if strings.Contains(strings.ToLower(msg.Content), searchTerm) ||
			strings.Contains(strings.ToLower(msg.AgentName), searchTerm) {
			m.searchResults = append(m.searchResults, i)
		}
	}

	// Set current index to first result if any found
	if len(m.searchResults) > 0 {
		m.currentSearchIndex = 0
		m.scrollToSearchResult()
	} else {
		m.currentSearchIndex = -1
	}
}

// scrollToSearchResult scrolls the viewport to show the current search result
func (m *Model) scrollToSearchResult() {
	if m.currentSearchIndex < 0 || m.currentSearchIndex >= len(m.searchResults) {
		return
	}

	// Get the message index
	msgIndex := m.searchResults[m.currentSearchIndex]

	// Calculate approximate line position
	// Each message takes roughly 4 lines (timestamp line + content + blank line + separator)
	linePos := msgIndex * 4

	// Scroll viewport to show this message
	// Try to position it in the middle of the viewport
	targetLine := linePos - (m.viewport.Height / 2)
	if targetLine < 0 {
		targetLine = 0
	}

	// Calculate the percentage position
	totalLines := len(m.messages) * 4
	if totalLines > 0 {
		percent := float64(targetLine) / float64(totalLines)
		m.viewport.SetYOffset(int(percent * float64(m.viewport.TotalLineCount())))
	}
}

func (m Model) startConversation() tea.Cmd {
	return func() tea.Msg {
		orchConfig := orchestrator.OrchestratorConfig{
			Mode:          orchestrator.ConversationMode(m.config.Orchestrator.Mode),
			TurnTimeout:   m.config.Orchestrator.TurnTimeout,
			MaxTurns:      m.config.Orchestrator.MaxTurns,
			ResponseDelay: m.config.Orchestrator.ResponseDelay,
			InitialPrompt: m.config.Orchestrator.InitialPrompt,
		}

		writer := &tuiWriter{
			messageChan: make(chan agent.Message, 100),
		}

		orch := orchestrator.NewOrchestrator(orchConfig, writer)

		for _, a := range m.agents {
			orch.AddAgent(a)
		}

		go func() {
			for range writer.messageChan {
				// Drain the channel
			}
		}()

		go func() {
			err := orch.Start(m.ctx)
			if err != nil {
				// Error is already logged by orchestrator, nothing to do here
				_ = err
			}
			close(writer.messageChan)
		}()

		return conversationDone{}
	}
}

type tuiWriter struct {
	messageChan chan agent.Message
}

func (w *tuiWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
