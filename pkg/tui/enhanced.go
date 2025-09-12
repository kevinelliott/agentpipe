package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
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

	// Styles
	agentColors map[string]lipgloss.Color
}

// Styles
var (
	// Panel styles
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63"))

	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	// Title styles
	enhancedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99")).
				Background(lipgloss.Color("235")).
				Padding(0, 1)

	// Agent list styles
	selectedAgentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("63")).
				Bold(true).
				Padding(0, 1)

	normalAgentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)

	// Modal styles
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2).
			Background(lipgloss.Color("235"))

	// Status bar styles
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	// Help styles
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))
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

func RunEnhanced(ctx context.Context, cfg *config.Config, agents []agent.Agent) error {
	// Create agent items for the list
	items := make([]list.Item, len(agents))
	agentColorMap := make(map[string]lipgloss.Color)

	for i, a := range agents {
		color := agentColors[i%len(agentColors)]
		agentColorMap[a.GetName()] = color
		items[i] = agentItem{
			agent: a,
			color: color,
		}
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
	ta.Focus()

	m := EnhancedModel{
		ctx:         ctx,
		config:      cfg,
		agents:      agents,
		agentList:   agentList,
		userInput:   ta,
		messages:    make([]agent.Message, 0),
		activePanel: conversationPanel,
		agentColors: agentColorMap,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m EnhancedModel) Init() tea.Cmd {
	return tea.Batch(
		m.startConversation(),
		textarea.Blink,
	)
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
			} else if m.activePanel == inputPanel && m.userInput.Value() != "" {
				// Send user message
				cmds = append(cmds, m.sendUserMessage())
			}

		case "up", "k":
			if m.activePanel == agentsPanel {
				m.agentList, _ = m.agentList.Update(msg)
			} else if m.activePanel == conversationPanel {
				m.conversation.LineUp(1)
			}

		case "down", "j":
			if m.activePanel == agentsPanel {
				m.agentList, _ = m.agentList.Update(msg)
			} else if m.activePanel == conversationPanel {
				m.conversation.LineDown(1)
			}

		case "pgup":
			if m.activePanel == conversationPanel {
				m.conversation.HalfViewUp()
			}

		case "pgdown":
			if m.activePanel == conversationPanel {
				m.conversation.HalfViewDown()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate panel dimensions with proper margins
		leftWidth := (msg.Width - 3) / 3 // Account for borders and spacing
		rightWidth := msg.Width - leftWidth - 3
		convHeight := msg.Height - 12 // Account for input, status bar, borders

		if !m.ready {
			// Initialize viewports with size
			m.conversation = viewport.New(rightWidth-2, convHeight)
			m.conversation.SetContent(m.renderConversation())

			m.agentList.SetSize(leftWidth-2, (msg.Height-6)/2)

			m.userInput.SetWidth(rightWidth - 4)
			m.userInput.SetHeight(3)

			m.ready = true
		} else {
			// Update sizes on resize
			m.conversation.Width = rightWidth - 2
			m.conversation.Height = convHeight
			m.conversation.SetContent(m.renderConversation())

			m.agentList.SetSize(leftWidth-2, (msg.Height-6)/2)

			m.userInput.SetWidth(rightWidth - 4)
		}

	case messageUpdate:
		m.messages = append(m.messages, msg.message)
		m.conversation.SetContent(m.renderConversation())
		m.conversation.GotoBottom()

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

	// Calculate panel dimensions with proper margins
	leftWidth := (m.width - 3) / 3
	rightWidth := m.width - leftWidth - 3

	// Render agent list panel
	agentsPanelStyle := inactivePanelStyle
	if m.activePanel == agentsPanel {
		agentsPanelStyle = activePanelStyle
	}

	agentsView := agentsPanelStyle.
		Width(leftWidth).
		Height(m.height / 2).
		Render(m.renderAgentList())

	// Render stats panel
	statsView := inactivePanelStyle.
		Width(leftWidth).
		Height(m.height/2 - 2).
		Render(m.renderStats())

	// Render conversation panel
	convPanelStyle := inactivePanelStyle
	if m.activePanel == conversationPanel {
		convPanelStyle = activePanelStyle
	}

	convView := convPanelStyle.
		Width(rightWidth).
		Height(m.height - 8).
		Render(m.conversation.View())

	// Render input panel
	inputPanelStyle := inactivePanelStyle
	if m.activePanel == inputPanel {
		inputPanelStyle = activePanelStyle
	}

	inputView := inputPanelStyle.
		Width(rightWidth).
		Height(5).
		Render(m.userInput.View())

	// Render status bar
	statusBar := m.renderStatusBar()

	// Combine all panels
	left := lipgloss.JoinVertical(lipgloss.Top,
		agentsView,
		statsView,
	)

	right := lipgloss.JoinVertical(lipgloss.Top,
		convView,
		inputView,
	)

	main := lipgloss.JoinHorizontal(lipgloss.Left, left, right)

	return lipgloss.JoinVertical(lipgloss.Top,
		main,
		statusBar,
	)
}

func (m *EnhancedModel) renderAgentList() string {
	var b strings.Builder

	b.WriteString(enhancedTitleStyle.Render("👥 Agents"))
	b.WriteString("\n\n")

	for i, a := range m.agents {
		color := m.agentColors[a.GetName()]

		prefix := "  "
		style := normalAgentStyle

		if m.activePanel == agentsPanel && i == m.selectedAgent {
			prefix = "▶ "
			style = selectedAgentStyle
		}

		badge := lipgloss.NewStyle().
			Background(color).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1).
			Render(a.GetName())

		b.WriteString(fmt.Sprintf("%s%s\n", prefix, badge))
		b.WriteString(style.Render(fmt.Sprintf("   %s\n", a.GetType())))
	}

	return b.String()
}

func (m *EnhancedModel) renderStats() string {
	var b strings.Builder

	b.WriteString(enhancedTitleStyle.Render("📊 Statistics"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Messages: %d\n", len(m.messages)))
	b.WriteString(fmt.Sprintf("  Agents: %d\n", len(m.agents)))
	b.WriteString(fmt.Sprintf("  Mode: %s\n", m.config.Orchestrator.Mode))

	if m.running {
		b.WriteString("\n  Status: 🟢 Running")
	} else {
		b.WriteString("\n  Status: 🔴 Stopped")
	}

	if m.userTurn {
		b.WriteString("\n  👤 User turn enabled")
	}

	return b.String()
}

func (m *EnhancedModel) renderConversation() string {
	var b strings.Builder

	for _, msg := range m.messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")

		// Get color for agent
		color := lipgloss.Color("244")
		if c, ok := m.agentColors[msg.AgentName]; ok {
			color = c
		}

		style := lipgloss.NewStyle().Foreground(color)

		if msg.Role == "system" {
			b.WriteString(fmt.Sprintf("[%s] 📢 %s\n\n", timestamp, msg.Content))
		} else if msg.AgentName == "User" {
			userStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("226")).
				Bold(true)
			b.WriteString(userStyle.Render(fmt.Sprintf("[%s] 👤 User\n", timestamp)))
			b.WriteString(fmt.Sprintf("  %s\n\n", msg.Content))
		} else {
			b.WriteString(style.Bold(true).Render(fmt.Sprintf("[%s] %s\n", timestamp, msg.AgentName)))
			b.WriteString(style.Render(fmt.Sprintf("  %s\n\n", msg.Content)))
		}
	}

	return b.String()
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

func (m *EnhancedModel) startConversation() tea.Cmd {
	return func() tea.Msg {
		// TODO: Integrate with orchestrator
		// For now, just return done
		return conversationDone{}
	}
}
