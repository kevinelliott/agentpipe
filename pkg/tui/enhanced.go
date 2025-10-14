package tui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/internal/version"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
)

type panel int

const (
	agentsPanel panel = iota
	conversationPanel
	inputPanel
)

type EnhancedModel struct {
	ctx    context.Context
	config *config.Config
	agents []agent.Agent
	orch   *orchestrator.Orchestrator

	// UI components
	agentList    list.Model
	conversation viewport.Model
	userInput    textarea.Model

	// State
	messages      []agent.Message
	activePanel   panel
	showModal     bool
	modalContent  string
	selectedAgent int
	width         int
	height        int
	ready         bool
	running       bool
	userTurn      bool
	err           error
	msgChan       <-chan agent.Message
	turnCount     int
	initialized   bool
	initializing  bool
	activeAgent   string             // Track which agent is currently responding
	chatLogger    *logger.ChatLogger // For logging conversations
	totalCost     float64            // Track total cost of conversation

	// Initialization params
	skipHealthCheck    bool
	healthCheckTimeout int
	configPath         string // Path to config file if used

	// Styles
	agentColors map[string]lipgloss.Color
}

// Styles
var (
	// Panel styles
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	// Input panel styles (no padding)
	activeInputPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63"))

	inactiveInputPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	// Title styles
	enhancedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99"))

	// Modal styles
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2).
			Background(lipgloss.Color("235"))

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Help styles
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))

	// Logo panel styles
	logoPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Align(lipgloss.Center)

	logoTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	logoInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Align(lipgloss.Center)
)

var agentColors = []lipgloss.Color{
	lipgloss.Color("63"),  // Blue
	lipgloss.Color("212"), // Pink
	lipgloss.Color("86"),  // Green
	lipgloss.Color("214"), // Orange
	lipgloss.Color("99"),  // Purple
	lipgloss.Color("51"),  // Cyan
	lipgloss.Color("226"), // Yellow
	lipgloss.Color("201"), // Magenta
}

type agentItem struct {
	agent agent.Agent
	color lipgloss.Color
}

func (i agentItem) FilterValue() string { return i.agent.GetName() }
func (i agentItem) Title() string       { return i.agent.GetName() }
func (i agentItem) Description() string {
	return fmt.Sprintf("Type: %s | ID: %s", i.agent.GetType(), i.agent.GetID())
}

func RunEnhanced(ctx context.Context, cfg *config.Config, agents []agent.Agent, skipHealthCheck bool, healthCheckTimeout int, configPath string) error {
	// Create agent items for the list
	var items []list.Item
	agentColorMap := make(map[string]lipgloss.Color)

	if agents != nil {
		// Agents already initialized
		items = make([]list.Item, len(agents))
		for i, a := range agents {
			color := agentColors[i%len(agentColors)]
			agentColorMap[a.GetName()] = color
			items[i] = agentItem{
				agent: a,
				color: color,
			}
		}
	} else {
		// Agents will be initialized after TUI starts
		items = []list.Item{}
		agents = []agent.Agent{}
	}

	// Create the agent list
	agentList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	agentList.Title = "Agents"
	agentList.SetShowStatusBar(false)
	agentList.SetFilteringEnabled(false)
	agentList.SetShowHelp(false)

	// Create the user input area
	ta := textarea.New()
	ta.Placeholder = "Type your message to join the conversation..."
	ta.ShowLineNumbers = false
	ta.Prompt = "> "
	ta.SetHeight(2) // Two line input

	// Remove all backgrounds from textarea
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ta.FocusedStyle.Text = lipgloss.NewStyle()

	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ta.BlurredStyle.Text = lipgloss.NewStyle()

	ta.Focus()

	// Create orchestrator configuration
	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ConversationMode(cfg.Orchestrator.Mode),
		TurnTimeout:   cfg.Orchestrator.TurnTimeout,
		MaxTurns:      cfg.Orchestrator.MaxTurns,
		ResponseDelay: cfg.Orchestrator.ResponseDelay,
		InitialPrompt: cfg.Orchestrator.InitialPrompt,
	}

	// Only set a default timeout if none was configured
	if orchConfig.TurnTimeout == 0 {
		orchConfig.TurnTimeout = 60 * time.Second // Default to 60 seconds for TUI
	}

	// Create a message channel for the orchestrator to send updates
	msgChan := make(chan agent.Message, 100)

	// Create orchestrator with a writer that sends to our channel
	orch := orchestrator.NewOrchestrator(orchConfig, &messageWriter{
		msgChan:        msgChan,
		buffer:         strings.Builder{},
		currentContent: strings.Builder{},
	})

	// Set up logging if enabled
	var chatLogger *logger.ChatLogger
	if cfg.Logging.Enabled {
		var err error
		chatLogger, err = logger.NewChatLogger(cfg.Logging.ChatLogDir, cfg.Logging.LogFormat, nil, cfg.Logging.ShowMetrics)
		if err != nil {
			// Log error but continue without logging
			fmt.Fprintf(os.Stderr, "Warning: Failed to create chat logger: %v\n", err)
		} else {
			orch.SetLogger(chatLogger)
		}
	}

	m := EnhancedModel{
		ctx:                ctx,
		config:             cfg,
		agents:             agents,
		orch:               orch,
		agentList:          agentList,
		userInput:          ta,
		messages:           make([]agent.Message, 0),
		activePanel:        conversationPanel,
		agentColors:        agentColorMap,
		msgChan:            msgChan,
		initialized:        len(agents) > 0,
		skipHealthCheck:    skipHealthCheck,
		healthCheckTimeout: healthCheckTimeout,
		chatLogger:         chatLogger,
		configPath:         configPath,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()

	// Close the message channel to signal cleanup
	close(msgChan)

	// Close the logger if it exists
	if chatLogger != nil {
		chatLogger.Close()
	}

	return err
}

func (m EnhancedModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textarea.Blink,
	}

	if !m.initialized {
		// Send initialization message first
		cmds = append(cmds, func() tea.Msg {
			return agentInitMsg{message: "🔍 Initializing agents..."}
		})
		// Start agent initialization
		cmds = append(cmds, m.initializeAgents())
	} else {
		// Agents already initialized, start conversation
		cmds = append(cmds, m.startConversation(), m.waitForMessage())
	}

	return tea.Batch(cmds...)
}

// initializeAgents initializes all agents and sends status updates
func (m EnhancedModel) initializeAgents() tea.Cmd {
	return func() tea.Msg {
		agentsList := make([]agent.Agent, 0)

		for _, agentCfg := range m.config.Agents {
			// Create agent
			a, err := agent.CreateAgent(agentCfg)
			if err != nil {
				return agentInitComplete{
					err: fmt.Errorf("failed to create agent %s: %w", agentCfg.Name, err),
				}
			}

			if !a.IsAvailable() {
				return agentInitComplete{
					err: fmt.Errorf("agent %s (type: %s) is not available - please run 'agentpipe doctor'", agentCfg.Name, agentCfg.Type),
				}
			}

			// Perform health check unless skipped
			if !m.skipHealthCheck {
				timeout := time.Duration(m.healthCheckTimeout) * time.Second
				if timeout == 0 {
					timeout = 5 * time.Second
				}

				healthCtx, cancel := context.WithTimeout(m.ctx, timeout)
				err = a.HealthCheck(healthCtx)
				cancel()

				if err != nil {
					return agentInitComplete{
						err: fmt.Errorf("agent %s failed health check: %w", agentCfg.Name, err),
					}
				}
			}

			agentsList = append(agentsList, a)
		}

		if len(agentsList) == 0 {
			return agentInitComplete{
				err: fmt.Errorf("no agents configured"),
			}
		}

		return agentInitComplete{
			agents: agentsList,
		}
	}
}

// waitForMessage polls for new messages from the orchestrator
func (m EnhancedModel) waitForMessage() tea.Cmd {
	return func() tea.Msg {
		// Check if there's a message waiting
		select {
		case msg := <-m.msgChan:
			return messageUpdate{message: msg}
		case <-time.After(100 * time.Millisecond):
			// No message, return a tick to check again
			return tickMsg{}
		}
	}
}

type tickMsg struct{}

type agentInitMsg struct {
	message string
}

type agentInitComplete struct {
	agents []agent.Agent
	err    error
}

func (m EnhancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		if m.showModal {
			if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
				m.showModal = false
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			// Cycle through panels
			m.activePanel = (m.activePanel + 1) % 3
			switch m.activePanel {
			case agentsPanel:
				m.agentList.SetDelegate(list.NewDefaultDelegate())
			case inputPanel:
				cmd := m.userInput.Focus()
				cmds = append(cmds, cmd)
			}

		case "ctrl+u":
			// Toggle user turn
			m.userTurn = !m.userTurn
			if m.userTurn {
				m.activePanel = inputPanel
				cmd := m.userInput.Focus()
				cmds = append(cmds, cmd)
			}

		case "enter":
			if m.activePanel == agentsPanel && len(m.agents) > 0 {
				// Show agent details modal
				selected := m.agentList.SelectedItem()
				if item, ok := selected.(agentItem); ok {
					m.showAgentModal(item.agent)
				}
			} else if m.activePanel == inputPanel {
				// Only send if there's actual content (not just the prompt)
				content := strings.TrimSpace(strings.TrimPrefix(m.userInput.Value(), ">"))
				if content != "" {
					// Send user message
					cmds = append(cmds, m.sendUserMessage())
					// Clear the input and reset cursor
					m.userInput.Reset()
					m.userInput.CursorStart()
				}
			}

		case "up", "k":
			if m.activePanel == agentsPanel {
				m.agentList, _ = m.agentList.Update(msg)
			} else if m.activePanel == conversationPanel {
				m.conversation.ScrollUp(1)
			}

		case "down", "j":
			if m.activePanel == agentsPanel {
				m.agentList, _ = m.agentList.Update(msg)
			} else if m.activePanel == conversationPanel {
				m.conversation.ScrollDown(1)
			}

		case "pgup":
			if m.activePanel == conversationPanel {
				m.conversation.HalfPageUp()
			}

		case "pgdown":
			if m.activePanel == conversationPanel {
				m.conversation.HalfPageDown()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate panel dimensions with room for borders (swapped: chat on left, agents on right)
		rightWidth := 33                         // Fixed width for agents/stats panels (reduced)
		leftWidth := msg.Width - rightWidth - 11 // Chat/input takes remaining width (increased by 1)

		// Account for topic panel if present
		topicHeight := 0
		if m.config.Orchestrator.InitialPrompt != "" {
			topicHeight = 4 // 3 for content + 1 for spacing (reduced by 2)
		}

		convHeight := msg.Height - 20 - topicHeight // Reduced by 1 more

		if !m.ready {
			// Initialize viewports with size (now using leftWidth for conversation)
			m.conversation = viewport.New(leftWidth-2, convHeight)
			m.conversation.SetContent(m.renderConversation())

			m.agentList.SetSize(rightWidth-2, (msg.Height-6)/2)

			m.userInput.SetWidth(leftWidth - 4)
			m.userInput.SetHeight(2)

			m.ready = true
		} else {
			// Update sizes on resize (swapped dimensions)
			m.conversation.Width = leftWidth - 2
			m.conversation.Height = convHeight
			m.conversation.SetContent(m.renderConversation())

			m.agentList.SetSize(rightWidth-2, (msg.Height-6)/2)

			m.userInput.SetWidth(leftWidth - 4)
		}

	case agentInitMsg:
		// Add initialization message to chat
		initMsg := agent.Message{
			AgentID:   "system",
			AgentName: "System",
			Content:   msg.message,
			Timestamp: time.Now().Unix(),
			Role:      "system",
		}
		m.messages = append(m.messages, initMsg)
		m.conversation.SetContent(m.renderConversation())
		m.conversation.GotoBottom()

	case agentInitComplete:
		if msg.err != nil {
			// Add error message to chat
			errMsg := agent.Message{
				AgentID:   "error",
				AgentName: "System",
				Content:   fmt.Sprintf("Failed to initialize agents: %v", msg.err),
				Timestamp: time.Now().Unix(),
				Role:      "system",
			}
			m.messages = append(m.messages, errMsg)
			m.conversation.SetContent(m.renderConversation())
			m.conversation.GotoBottom()
			m.err = msg.err
			return m, nil
		}

		// Successfully initialized agents
		m.agents = msg.agents
		m.initialized = true
		m.initializing = false

		// Update agent list
		items := make([]list.Item, len(m.agents))
		for i, a := range m.agents {
			color := agentColors[i%len(agentColors)]
			m.agentColors[a.GetName()] = color
			items[i] = agentItem{
				agent: a,
				color: color,
			}
		}
		m.agentList.SetItems(items)

		// Add success message
		successMsg := agent.Message{
			AgentID:   "info",
			AgentName: "System",
			Content:   fmt.Sprintf("✅ All %d agents initialized successfully", len(m.agents)),
			Timestamp: time.Now().Unix(),
			Role:      "system",
		}
		m.messages = append(m.messages, successMsg)
		m.conversation.SetContent(m.renderConversation())
		m.conversation.GotoBottom()

		// Don't add agents here - they'll be added in startConversation
		// Mark as running before starting conversation
		m.running = true
		// Start the conversation
		cmds = append(cmds, m.startConversation(), m.waitForMessage())

	case messageUpdate:
		if msg.message.Role == "active" {
			// This is just an indicator that an agent is actively typing
			m.activeAgent = msg.message.AgentName
		} else {
			// Regular message
			m.messages = append(m.messages, msg.message)

			// Log the message if logging is enabled
			if m.chatLogger != nil {
				m.chatLogger.LogMessage(msg.message)
			}

			// Track turn count and cost for agent messages (not system/error messages)
			if msg.message.Role == "agent" {
				m.turnCount++
				// Clear active agent when message is complete
				if msg.message.AgentName == m.activeAgent {
					m.activeAgent = ""
				}
				// Accumulate cost if metrics are available
				if msg.message.Metrics != nil && msg.message.Metrics.Cost > 0 {
					m.totalCost += msg.message.Metrics.Cost
				}
			}
			// If this is the "Starting AgentPipe conversation" message, mark as running
			if strings.Contains(msg.message.Content, "Starting AgentPipe conversation") {
				m.running = true
			}
			m.conversation.SetContent(m.renderConversation())
			m.conversation.GotoBottom()
		}
		// Continue polling for messages
		cmds = append(cmds, m.waitForMessage())

	case tickMsg:
		// Continue polling for messages if still running or if we have agents
		if m.running || len(m.agents) > 0 {
			cmds = append(cmds, m.waitForMessage())
		}

	case conversationDone:
		m.running = false

	case errMsg:
		m.err = msg.err
		m.running = false
	}

	// Update sub-components
	if m.ready && !m.showModal {
		if m.activePanel == inputPanel {
			var cmd tea.Cmd
			m.userInput, cmd = m.userInput.Update(msg)
			cmds = append(cmds, cmd)
		}

		if m.activePanel == conversationPanel {
			var cmd tea.Cmd
			m.conversation, cmd = m.conversation.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m EnhancedModel) View() string {
	if !m.ready {
		return "Initializing AgentPipe TUI..."
	}

	// Show modal if active
	if m.showModal {
		return m.renderModal()
	}

	// Calculate panel dimensions with room for borders (swapped: chat on left, agents on right)
	rightWidth := 33                       // Fixed width for agents/stats panels (reduced)
	leftWidth := m.width - rightWidth - 11 // Chat/input takes remaining width (increased by 1)

	// Render topic panel (new panel above conversation)
	topicView := ""
	topicHeight := 0
	if m.config.Orchestrator.InitialPrompt != "" {
		topicHeight = 3 // Fixed height for topic panel (reduced by 2)
		topicPanelStyle := inactivePanelStyle

		// Format topic content - limit to 2 lines
		topicTitle := lipgloss.NewStyle().Bold(true).Render("📝 Topic")

		// Truncate topic to fit in 2 lines (accounting for width)
		maxWidth := leftWidth - 4 // Account for padding
		prompt := m.config.Orchestrator.InitialPrompt
		lines := wrapText(prompt, maxWidth)
		lineArray := strings.Split(lines, "\n")
		if len(lineArray) > 2 {
			// Take first 2 lines and add ellipsis
			prompt = lineArray[0] + "\n" + lineArray[1] + "..."
		} else {
			prompt = lines
		}

		topicContent := fmt.Sprintf("%s\n%s", topicTitle, prompt)

		topicView = topicPanelStyle.
			Width(leftWidth).
			Height(topicHeight).
			Render(topicContent)
	}

	// Render conversation panel (now on left, below topic)
	convPanelStyle := inactivePanelStyle
	if m.activePanel == conversationPanel {
		convPanelStyle = activePanelStyle
	}

	convView := convPanelStyle.
		Width(leftWidth).
		Height(m.height - 20 - topicHeight - 1). // Reduced by 1 more
		Render(m.conversation.View())

	// Render input panel (now on left)
	inputPanelStyle := inactiveInputPanelStyle
	if m.activePanel == inputPanel {
		inputPanelStyle = activeInputPanelStyle
	}

	// Render input with proper formatting
	inputContent := m.userInput.View()
	// Ensure we show > prompts on empty lines
	if strings.TrimSpace(inputContent) == "" || inputContent == "> " {
		inputContent = "> \n"
	}

	inputView := inputPanelStyle.
		Width(leftWidth).
		Height(2).
		Render(inputContent)

	// Render agent list panel (now on right)
	agentsPanelStyle := inactivePanelStyle
	if m.activePanel == agentsPanel {
		agentsPanelStyle = activePanelStyle
	}

	// Calculate heights for 3 panels on the right
	// Make stats panel smaller
	totalRightHeight := m.height - 15
	agentsPanelHeight := totalRightHeight / 3
	configPanelHeight := totalRightHeight / 3
	statsPanelHeight := totalRightHeight - agentsPanelHeight - configPanelHeight - 4 // Reduced by 3 more

	agentsView := agentsPanelStyle.
		Width(rightWidth).
		Height(agentsPanelHeight).
		Render(m.renderAgentList())

	// Render config panel (middle right)
	configView := inactivePanelStyle.
		Width(rightWidth).
		Height(configPanelHeight).
		Render(m.renderConfig())

	// Render stats panel (bottom right, smaller)
	statsView := inactivePanelStyle.
		Width(rightWidth).
		Height(statsPanelHeight).
		Render(m.renderStats())

	// Render status bar
	statusBar := m.renderStatusBar()

	// Combine all panels (swapped: chat/input on left, agents/stats on right)
	leftPanels := []string{}
	if topicView != "" {
		leftPanels = append(leftPanels, topicView)
	}
	leftPanels = append(leftPanels, convView, inputView)

	left := lipgloss.JoinVertical(lipgloss.Top, leftPanels...)

	right := lipgloss.JoinVertical(lipgloss.Top,
		agentsView,
		configView,
		statsView,
	)

	main := lipgloss.JoinHorizontal(lipgloss.Left, left, right)

	// Render logo panel at the top
	logoView := m.renderLogo()

	// Ensure the final output fits within terminal bounds
	return lipgloss.NewStyle().
		MaxWidth(m.width - 6).
		MaxHeight(m.height - 1).
		Render(lipgloss.JoinVertical(lipgloss.Top,
			logoView,
			main,
			statusBar,
		))
}

func (m *EnhancedModel) renderAgentList() string {
	var b strings.Builder

	b.WriteString(enhancedTitleStyle.Render("👥 Agents"))
	b.WriteString("\n\n") // Add blank line after title

	// Calculate available width for alignment
	availableWidth := 30 // Adjust based on panel width

	for i, a := range m.agents {
		color := m.agentColors[a.GetName()]

		// Create colored name style
		nameStyle := lipgloss.NewStyle().
			Foreground(color).
			Bold(true)

		// Type style in gray
		typeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

		// Selection indicator
		indicator := ""
		if m.activePanel == agentsPanel && i == m.selectedAgent {
			indicator = "▶ "
			nameStyle = nameStyle.Background(lipgloss.Color("235"))
		}

		// Active indicator (green dot when agent is responding, grey when inactive)
		activeColor := lipgloss.Color("240") // Grey color for inactive
		if m.activeAgent == a.GetName() {
			activeColor = lipgloss.Color("82") // Green color for active
		}
		statusDot := lipgloss.NewStyle().Foreground(activeColor).Render("●")

		// Create left-aligned name and right-aligned type
		name := nameStyle.Render(a.GetName())
		agentType := typeStyle.Render(a.GetType())

		// Calculate spacing
		nameLen := len(a.GetName()) + len(indicator) + 2 // +2 for status dot and space
		typeLen := len(a.GetType())
		spaces := availableWidth - nameLen - typeLen
		if spaces < 1 {
			spaces = 1
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s%s\n",
			indicator,
			statusDot,
			name,
			strings.Repeat(" ", spaces),
			agentType))
	}

	return b.String()
}

func (m *EnhancedModel) renderConfig() string {
	var b strings.Builder

	b.WriteString(enhancedTitleStyle.Render("⚙️  Config"))
	b.WriteString("\n\n") // Add blank line after title

	// Calculate available width for alignment
	availableWidth := 30

	// Show config file if used
	if m.configPath != "" {
		// Truncate long paths
		path := m.configPath
		if len(path) > 28 {
			path = "..." + path[len(path)-25:]
		}
		b.WriteString(fmt.Sprintf("File: %s\n\n", path))
	}

	// Format with left/right alignment
	items := []struct {
		label string
		value string
	}{
		{"Mode:", m.config.Orchestrator.Mode},
		{"Max Turns:", fmt.Sprintf("%d", m.config.Orchestrator.MaxTurns)},
		{"Timeout:", fmt.Sprintf("%ds", int(m.config.Orchestrator.TurnTimeout.Seconds()))},
		{"Delay:", fmt.Sprintf("%ds", int(m.config.Orchestrator.ResponseDelay.Seconds()))},
	}

	for _, item := range items {
		spaces := availableWidth - len(item.label) - len(item.value)
		if spaces < 1 {
			spaces = 1
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", item.label, strings.Repeat(" ", spaces), item.value))
	}

	if m.config.Logging.Enabled {
		b.WriteString("\nLogging:                    ✅")
		if m.config.Logging.ShowMetrics {
			b.WriteString("\nMetrics:                    ✅")
		}
	} else {
		b.WriteString("\nLogging:                    ❌")
	}

	return b.String()
}

func (m *EnhancedModel) renderStats() string {
	var b strings.Builder

	b.WriteString(enhancedTitleStyle.Render("📊 Statistics"))
	b.WriteString("\n\n") // Add blank line after title

	// Calculate available width for alignment
	availableWidth := 30

	// Count connected agents (those that are initialized)
	connectedAgents := len(m.agents)
	configuredAgents := len(m.config.Agents)

	// Format turns display
	turnsDisplay := fmt.Sprintf("%d/%d", m.turnCount, m.config.Orchestrator.MaxTurns)
	if m.config.Orchestrator.MaxTurns == 0 {
		turnsDisplay = fmt.Sprintf("%d/∞", m.turnCount)
	}

	// Format with left/right alignment
	items := []struct {
		label string
		value string
	}{
		{"Messages:", fmt.Sprintf("%d", len(m.messages))},
		{"Agents:", fmt.Sprintf("%d/%d", connectedAgents, configuredAgents)},
		{"Turns:", turnsDisplay},
		{"Total Cost:", fmt.Sprintf("$%.4f", m.totalCost)},
	}

	for _, item := range items {
		spaces := availableWidth - len(item.label) - len(item.value)
		if spaces < 1 {
			spaces = 1
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", item.label, strings.Repeat(" ", spaces), item.value))
	}

	// Status with emoji
	status := "🔴 Stopped"
	if m.running {
		status = "🟢 Running"
	}
	spaces := availableWidth - 7 - len(status) // "Status:" is 7 chars
	if spaces < 1 {
		spaces = 1
	}
	b.WriteString(fmt.Sprintf("\nStatus:%s%s", strings.Repeat(" ", spaces), status))

	if m.userTurn {
		b.WriteString("\n👤 User turn enabled")
	}

	return b.String()
}

func (m *EnhancedModel) renderConversation() string {
	var b strings.Builder

	// Calculate available width for text (account for padding and timestamp)
	textWidth := m.conversation.Width - 4 // Leave room for padding
	if textWidth < 20 {
		textWidth = 20 // Minimum width
	}

	lastSpeaker := ""

	for i, msg := range m.messages {
		// Don't show the initial prompt in the conversation since we have a Topic panel
		if msg.Role == "system" && m.config.Orchestrator.InitialPrompt != "" &&
			strings.Contains(msg.Content, m.config.Orchestrator.InitialPrompt) {
			continue // Skip showing the initial prompt in the conversation
		}

		// Determine the display name for this message
		displayName := ""
		if msg.Role == "system" {
			if msg.AgentID == "error" {
				displayName = "System Error"
			} else if msg.AgentID == "info" {
				displayName = "System Info"
			} else {
				displayName = "System Info" // Changed from "System" to "System Info"
			}
		} else if msg.AgentName == "User" {
			displayName = "User"
		} else {
			displayName = msg.AgentName
		}

		// Only show header if speaker changed
		if displayName != lastSpeaker {
			// Add newline before header (except for first message)
			if i > 0 {
				b.WriteString("\n")
			}
			timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")

			// Get color for agent
			color := lipgloss.Color("244")
			if c, ok := m.agentColors[msg.AgentName]; ok {
				color = c
			}

			if msg.Role == "system" {
				if msg.AgentID == "error" {
					errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
					b.WriteString(fmt.Sprintf("[%s] ", timestamp))
					b.WriteString(errorStyle.Render(displayName))
				} else if msg.AgentID == "info" {
					infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // Blue
					b.WriteString(fmt.Sprintf("[%s] ", timestamp))
					b.WriteString(infoStyle.Render(displayName))
				} else {
					systemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // Grey
					b.WriteString(fmt.Sprintf("[%s] ", timestamp))
					b.WriteString(systemStyle.Render(displayName))
				}
			} else if msg.AgentName == "User" {
				userStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("226")).
					Bold(true)
				b.WriteString(fmt.Sprintf("[%s] ", timestamp))
				b.WriteString(userStyle.Render("👤 " + displayName))
			} else {
				// Agent messages
				style := lipgloss.NewStyle().Foreground(color).Bold(true)
				b.WriteString(fmt.Sprintf("[%s] ", timestamp))
				b.WriteString(style.Render(displayName))
			}

			// Add metrics if available and enabled (only for agents, not system messages)
			if msg.Role != "system" && m.config.Logging.ShowMetrics && msg.Metrics != nil {
				seconds := msg.Metrics.Duration.Seconds()
				metricsStr := fmt.Sprintf(" (%.1fs, %d tokens, $%.4f)",
					seconds,
					msg.Metrics.TotalTokens,
					msg.Metrics.Cost)
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(metricsStr))
			}
			b.WriteString("\n")

			lastSpeaker = displayName
		}

		// Add the message content
		wrappedContent := wrapText(msg.Content, textWidth)

		// Apply color to content for system messages
		if msg.Role == "system" {
			if msg.AgentID == "error" {
				errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
				b.WriteString(errorStyle.Render(wrappedContent))
			} else if msg.AgentID == "info" {
				infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
				b.WriteString(infoStyle.Render(wrappedContent))
			} else {
				b.WriteString(wrappedContent)
			}
		} else {
			b.WriteString(wrappedContent)
		}

		// Add single newline after content (for same speaker continuation)
		// The spacing for different speakers is handled by the header
		if i < len(m.messages)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// wrapText wraps text to fit within the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		for len(line) > width {
			// Find last space before width
			cutPoint := width
			for i := width - 1; i > 0; i-- {
				if line[i] == ' ' {
					cutPoint = i
					break
				}
			}

			result = append(result, line[:cutPoint])
			line = strings.TrimSpace(line[cutPoint:])
		}
		if len(line) > 0 {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (m *EnhancedModel) renderLogo() string {
	logo := `    _    ____ _____ _   _ _____ ____ ___ ____  _____ 
   / \  / ___| ____| \ | |_   _|  _ \_ _|  _ \| ____|
  / _ \| |  _|  _| |  \| | | | | |_) | || |_) |  _|  
 / ___ \ |_| | |___| |\  | | | |  __/| ||  __/| |___ 
/_/   \_\____|_____|_| \_| |_| |_|  |___|_|   |_____|`

	versionInfo := fmt.Sprintf("%s // https://github.com/kevinelliott/agentpipe", version.Version)

	content := lipgloss.JoinVertical(lipgloss.Center,
		logoTextStyle.Render(logo),
		"", // Add blank line
		logoInfoStyle.Render(versionInfo),
	)

	return logoPanelStyle.
		Width(m.width - 9).
		Height(8).
		Render(content)
}

func (m *EnhancedModel) renderStatusBar() string {
	help := []string{
		helpKeyStyle.Render("Tab") + helpDescStyle.Render(" Switch panel"),
		helpKeyStyle.Render("↑↓") + helpDescStyle.Render(" Navigate"),
		helpKeyStyle.Render("Enter") + helpDescStyle.Render(" Select/Send"),
		helpKeyStyle.Render("Ctrl+U") + helpDescStyle.Render(" User mode"),
		helpKeyStyle.Render("Q") + helpDescStyle.Render(" Quit"),
	}

	return statusBarStyle.
		Width(m.width).
		Render(strings.Join(help, " • "))
}

func (m *EnhancedModel) showAgentModal(a agent.Agent) {
	m.showModal = true

	var b strings.Builder
	b.WriteString(enhancedTitleStyle.Render(fmt.Sprintf("Agent Details: %s", a.GetName())))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("ID: %s\n", a.GetID()))
	b.WriteString(fmt.Sprintf("Type: %s\n", a.GetType()))
	b.WriteString(fmt.Sprintf("Name: %s\n", a.GetName()))
	b.WriteString("\n")
	b.WriteString("Status: ")
	if a.IsAvailable() {
		b.WriteString("✅ Available")
	} else {
		b.WriteString("❌ Unavailable")
	}
	b.WriteString("\n\n")
	b.WriteString("Press ESC or Enter to close")

	m.modalContent = b.String()
}

func (m *EnhancedModel) renderModal() string {
	modal := modalStyle.
		Width(50).
		Align(lipgloss.Center).
		Render(m.modalContent)

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

func (m *EnhancedModel) sendUserMessage() tea.Cmd {
	return func() tea.Msg {
		text := m.userInput.Value()
		m.userInput.Reset()
		m.userInput.CursorStart()

		msg := agent.Message{
			AgentID:   "user",
			AgentName: "User",
			Content:   text,
			Timestamp: time.Now().Unix(),
			Role:      "user",
		}

		return messageUpdate{message: msg}
	}
}

// messageWriter implements io.Writer to capture orchestrator output
type messageWriter struct {
	msgChan        chan<- agent.Message
	buffer         strings.Builder
	currentAgent   string                 // Track current speaking agent
	currentContent strings.Builder        // Accumulate content for current agent
	currentMetrics *agent.ResponseMetrics // Metrics for current message
	droppedCount   int                    // Track number of dropped messages
}

func (w *messageWriter) Write(p []byte) (n int, err error) {
	content := string(p)
	w.buffer.WriteString(content)

	// Process complete lines
	lines := strings.Split(w.buffer.String(), "\n")
	w.buffer.Reset()

	// Keep incomplete line in buffer
	if len(lines) > 0 && !strings.HasSuffix(content, "\n") {
		w.buffer.WriteString(lines[len(lines)-1])
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if this line starts a new message
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			// First, send any accumulated content from previous agent
			w.flushCurrentMessage()

			idx := strings.Index(line, "]")
			if idx > 0 {
				agentInfo := strings.TrimSpace(line[1:idx])
				messageContent := strings.TrimSpace(line[idx+1:])

				// Parse agent name and metrics if present (format: "AgentName|XXXms|XXXt|X.XXXX")
				var agentName string
				var metrics *agent.ResponseMetrics
				if strings.Contains(agentInfo, "|") {
					parts := strings.Split(agentInfo, "|")
					agentName = parts[0]
					if len(parts) >= 3 {
						// Parse metrics
						metrics = &agent.ResponseMetrics{}
						// Parse duration (e.g., "123ms")
						if strings.HasSuffix(parts[1], "ms") {
							if ms, err := strconv.Atoi(strings.TrimSuffix(parts[1], "ms")); err == nil {
								metrics.Duration = time.Duration(ms) * time.Millisecond
							}
						}
						// Parse tokens (e.g., "456t")
						if strings.HasSuffix(parts[2], "t") {
							if tokens, err := strconv.Atoi(strings.TrimSuffix(parts[2], "t")); err == nil {
								metrics.TotalTokens = tokens
							}
						}
						// Parse cost if available (e.g., "0.0012")
						if len(parts) >= 4 {
							if cost, err := strconv.ParseFloat(parts[3], 64); err == nil {
								metrics.Cost = cost
							}
						}
					}
				} else {
					agentName = agentInfo
				}

				if agentName == "System" || agentName == "Error" || agentName == "Info" {
					// Handle system messages immediately
					var msg agent.Message
					msg.Timestamp = time.Now().Unix()

					if agentName == "System" {
						msg.AgentID = "system"
						msg.AgentName = "System"
						msg.Content = messageContent
						msg.Role = "system"
					} else if agentName == "Error" {
						msg.AgentID = "error"
						msg.AgentName = "Error"
						// Parse error message to extract agent name if present
						if strings.Contains(messageContent, "Agent") && strings.Contains(messageContent, "failed:") {
							if strings.Contains(messageContent, "context deadline exceeded") {
								parts := strings.Split(messageContent, " failed:")
								if len(parts) > 0 {
									msg.Content = fmt.Sprintf("❌ %s timed out - response took too long", parts[0])
								} else {
									msg.Content = "❌ " + messageContent
								}
							} else {
								msg.Content = "❌ " + messageContent
							}
						} else {
							msg.Content = "❌ Error: " + messageContent
						}
						msg.Role = "system"
					} else if agentName == "Info" {
						msg.AgentID = "info"
						msg.AgentName = "Info"
						msg.Content = "ℹ️ " + messageContent
						msg.Role = "system"
					}

					if msg.Content != "" {
						select {
						case w.msgChan <- msg:
						default:
							// Channel full, drop message
							w.droppedCount++
							fmt.Fprintf(os.Stderr, "Warning: Message channel full, dropped message (total dropped: %d)\n", w.droppedCount)
						}
					}
				} else {
					// This is an agent message, start accumulating
					w.currentAgent = agentName
					w.currentMetrics = metrics
					w.currentContent.Reset()
					if messageContent != "" {
						w.currentContent.WriteString(messageContent)
					}
				}
			}
		} else if line != "" && w.currentAgent != "" {
			// This is a continuation of the current agent's message
			if w.currentContent.Len() > 0 {
				w.currentContent.WriteString("\n")
			}
			w.currentContent.WriteString(line)

			// Send an update that this agent is actively typing
			if w.currentAgent != "" {
				activeMsg := agent.Message{
					AgentID:   "_active",
					AgentName: w.currentAgent,
					Content:   "",
					Timestamp: time.Now().Unix(),
					Role:      "active",
				}
				select {
				case w.msgChan <- activeMsg:
				default:
				}
			}
		} else if line == "" && w.currentAgent != "" {
			// Empty line within an agent's message - preserve it
			if w.currentContent.Len() > 0 {
				w.currentContent.WriteString("\n\n")
			}
		}
	}

	// Check if we should flush (e.g., if we see certain patterns that indicate end of message)
	// This helps ensure messages are sent promptly
	if strings.Contains(content, "\n\n") || strings.HasSuffix(content, "\n") {
		w.flushCurrentMessage()
	}

	return len(p), nil
}

// flushCurrentMessage sends the accumulated message for the current agent
func (w *messageWriter) flushCurrentMessage() {
	if w.currentAgent != "" && w.currentContent.Len() > 0 {
		msg := agent.Message{
			AgentID:   w.currentAgent,
			AgentName: w.currentAgent,
			Content:   strings.TrimSpace(w.currentContent.String()),
			Timestamp: time.Now().Unix(),
			Role:      "agent",
			Metrics:   w.currentMetrics,
		}

		select {
		case w.msgChan <- msg:
		default:
			// Channel full, drop message
			w.droppedCount++
			fmt.Fprintf(os.Stderr, "Warning: Message channel full, dropped message (total dropped: %d)\n", w.droppedCount)
		}

		w.currentAgent = ""
		w.currentContent.Reset()
		w.currentMetrics = nil
	}
}

func (m *EnhancedModel) startConversation() tea.Cmd {
	return func() tea.Msg {
		// Add initial system message
		startMsg := agent.Message{
			AgentID:   "system",
			AgentName: "System",
			Content:   fmt.Sprintf("🚀 Starting AgentPipe conversation in %s mode...", m.config.Orchestrator.Mode),
			Timestamp: time.Now().Unix(),
			Role:      "system",
		}

		// Add agents to orchestrator and announce them
		for _, a := range m.agents {
			m.orch.AddAgent(a)
		}

		// Create a done channel to track orchestrator completion
		orchDone := make(chan struct{})

		// Start the orchestrator in a goroutine
		go func() {
			defer close(orchDone)

			// Use a longer timeout context for the entire conversation
			orchCtx, cancel := context.WithTimeout(m.ctx, 10*time.Minute)
			defer cancel()

			if err := m.orch.Start(orchCtx); err != nil {
				// Log error to stderr for debugging
				fmt.Fprintf(os.Stderr, "Orchestrator error: %v\n", err)
			}
			// Mark as not running when done
			m.running = false
		}()

		// Wait for orchestrator to finish with a timeout on TUI exit
		// This goroutine will clean up when the orchestrator completes
		go func() {
			select {
			case <-orchDone:
				// Orchestrator finished normally
			case <-m.ctx.Done():
				// Context canceled (TUI exiting), wait briefly for orchestrator
				select {
				case <-orchDone:
					// Orchestrator finished during grace period
				case <-time.After(2 * time.Second):
					// Grace period expired, orchestrator will be canceled by its own context
					fmt.Fprintf(os.Stderr, "Warning: Orchestrator did not finish within grace period\n")
				}
			}
		}()

		// Return the initial startup message
		return messageUpdate{message: startMsg}
	}
}
