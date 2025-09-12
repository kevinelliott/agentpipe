package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
)

type Model struct {
	ctx      context.Context
	config   *config.Config
	agents   []agent.Agent
	messages []agent.Message
	viewport viewport.Model
	textarea textarea.Model
	width    int
	height   int
	ready    bool
	running  bool
	err      error
}

type messageUpdate struct {
	message agent.Message
}

type conversationDone struct{}

type errMsg struct {
	err error
}

func Run(ctx context.Context, cfg *config.Config, agents []agent.Agent) error {
	m := Model{
		ctx:      ctx,
		config:   cfg,
		agents:   agents,
		messages: make([]agent.Message, 0),
		running:  false,
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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
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

	help := helpStyle.Render("Ctrl+C: Quit | Ctrl+S: Start | Ctrl+P: Pause/Resume | ↑↓: Scroll")
	b.WriteString(help)

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return b.String()
}

func (m Model) renderMessages() string {
	var b strings.Builder

	for _, msg := range m.messages {
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
