package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/kevinelliott/agentpipe/pkg/adapters"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
	"github.com/kevinelliott/agentpipe/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configPath         string
	agents             []string
	mode               string
	maxTurns           int
	turnTimeout        int
	responseDelay      int
	initialPrompt      string
	useTUI             bool
	healthCheckTimeout int
	chatLogDir         string
	disableLogging     bool
	showMetrics        bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a conversation between AI agents",
	Long: `Start a conversation between multiple AI agents. You can specify agents
directly via command line flags or use a YAML configuration file.`,
	Run: runConversation,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to YAML configuration file")
	runCmd.Flags().StringSliceVarP(&agents, "agents", "a", []string{}, "Agents to use (e.g., claude:Assistant1,gemini:Assistant2)")
	runCmd.Flags().StringVarP(&mode, "mode", "m", "round-robin", "Conversation mode (round-robin, reactive, free-form)")
	runCmd.Flags().IntVar(&maxTurns, "max-turns", 10, "Maximum number of conversation turns")
	runCmd.Flags().IntVar(&turnTimeout, "timeout", 30, "Turn timeout in seconds")
	runCmd.Flags().IntVar(&responseDelay, "delay", 1, "Delay between responses in seconds")
	runCmd.Flags().StringVarP(&initialPrompt, "prompt", "p", "", "Initial prompt to start the conversation")
	runCmd.Flags().BoolVarP(&useTUI, "tui", "t", false, "Use TUI interface")
	runCmd.Flags().Bool("skip-health-check", false, "Skip agent health checks (not recommended)")
	runCmd.Flags().IntVar(&healthCheckTimeout, "health-check-timeout", 2, "Health check timeout in seconds")
	runCmd.Flags().StringVar(&chatLogDir, "log-dir", "", "Directory to save chat logs (default: ~/.agentpipe/chats)")
	runCmd.Flags().BoolVar(&disableLogging, "no-log", false, "Disable chat logging")
	runCmd.Flags().BoolVar(&showMetrics, "metrics", false, "Show response metrics (duration, tokens, cost)")
}

func runConversation(cobraCmd *cobra.Command, args []string) {
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else if len(agents) > 0 {
		cfg = config.NewDefaultConfig()
		for i, agentSpec := range agents {
			agentCfg, err := parseAgentSpec(agentSpec, i)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing agent spec: %v\n", err)
				os.Exit(1)
			}
			cfg.Agents = append(cfg.Agents, agentCfg)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: Either --config or --agents must be specified\n")
		os.Exit(1)
	}

	if mode != "" {
		cfg.Orchestrator.Mode = mode
	}
	if maxTurns > 0 {
		cfg.Orchestrator.MaxTurns = maxTurns
	}
	if turnTimeout > 0 {
		cfg.Orchestrator.TurnTimeout = time.Duration(turnTimeout) * time.Second
	}
	if responseDelay > 0 {
		cfg.Orchestrator.ResponseDelay = time.Duration(responseDelay) * time.Second
	}
	if initialPrompt != "" {
		cfg.Orchestrator.InitialPrompt = initialPrompt
	}

	// Apply CLI overrides for logging
	if disableLogging {
		cfg.Logging.Enabled = false
	}
	if chatLogDir != "" {
		cfg.Logging.ChatLogDir = chatLogDir
		cfg.Logging.Enabled = true
	}
	if showMetrics {
		cfg.Logging.ShowMetrics = true
	}

	if err := startConversation(cobraCmd, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseAgentSpec(spec string, index int) (agent.AgentConfig, error) {
	var agentType, name string

	if n := len(spec); n > 0 {
		for i := 0; i < n; i++ {
			if spec[i] == ':' {
				agentType = spec[:i]
				if i+1 < n {
					name = spec[i+1:]
				}
				break
			}
		}
	}

	if agentType == "" {
		agentType = spec
	}

	if name == "" {
		name = fmt.Sprintf("%s-agent-%d", agentType, index+1)
	}

	return agent.AgentConfig{
		ID:   fmt.Sprintf("%s-%d", agentType, index),
		Type: agentType,
		Name: name,
	}, nil
}

func startConversation(cmd *cobra.Command, cfg *config.Config) error {
	agentsList := make([]agent.Agent, 0)

	verbose := viper.GetBool("verbose")

	fmt.Println("🔍 Initializing agents...")

	for _, agentCfg := range cfg.Agents {
		if verbose {
			fmt.Printf("  Creating agent %s (type: %s)...\n", agentCfg.Name, agentCfg.Type)
		}

		a, err := agent.CreateAgent(agentCfg)
		if err != nil {
			return fmt.Errorf("failed to create agent %s: %w", agentCfg.Name, err)
		}

		if !a.IsAvailable() {
			return fmt.Errorf("agent %s (type: %s) is not available - please run 'agentpipe doctor'", agentCfg.Name, agentCfg.Type)
		}

		// Perform health check unless skipped
		skipHealthCheck, err := cmd.Flags().GetBool("skip-health-check")
		if err != nil {
			skipHealthCheck = false
		}
		if !skipHealthCheck {
			if verbose {
				fmt.Printf("  Checking health of %s...\n", agentCfg.Name)
			}

			timeout := time.Duration(healthCheckTimeout) * time.Second
			if timeout == 0 {
				timeout = 2 * time.Second
			}

			healthCtx, cancel := context.WithTimeout(context.Background(), timeout)
			err = a.HealthCheck(healthCtx)
			cancel()

			if err != nil {
				fmt.Printf("  ⚠️  Health check failed for %s: %v\n", agentCfg.Name, err)
				fmt.Printf("  Troubleshooting tips:\n")
				fmt.Printf("    - Make sure the %s CLI is properly installed and configured\n", agentCfg.Type)
				fmt.Printf("    - Try running the CLI manually to check if it works\n")
				fmt.Printf("    - Check if API keys or authentication is required\n")
				fmt.Printf("    - Use --skip-health-check to bypass this check (not recommended)\n")
				if verbose {
					fmt.Printf("    - Full error: %v\n", err)
				}
				return fmt.Errorf("agent %s failed health check", agentCfg.Name)
			}

			if verbose {
				fmt.Printf("  ✅ Agent %s is ready\n", agentCfg.Name)
			}
		} else if verbose {
			fmt.Printf("  ⚠️  Skipping health check for %s\n", agentCfg.Name)
		}

		agentsList = append(agentsList, a)
	}

	if len(agentsList) == 0 {
		return fmt.Errorf("no agents configured")
	}

	fmt.Printf("✅ All %d agents initialized successfully\n\n", len(agentsList))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nInterrupted. Shutting down...")
		cancel()
	}()

	if useTUI {
		// Use enhanced TUI
		return tui.RunEnhanced(ctx, cfg, agentsList)
	}

	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ConversationMode(cfg.Orchestrator.Mode),
		TurnTimeout:   cfg.Orchestrator.TurnTimeout,
		MaxTurns:      cfg.Orchestrator.MaxTurns,
		ResponseDelay: cfg.Orchestrator.ResponseDelay,
		InitialPrompt: cfg.Orchestrator.InitialPrompt,
	}

	// Create logger if enabled
	var chatLogger *logger.ChatLogger
	if cfg.Logging.Enabled {
		var err error
		chatLogger, err = logger.NewChatLogger(cfg.Logging.ChatLogDir, cfg.Logging.LogFormat, os.Stdout, cfg.Logging.ShowMetrics)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create chat logger: %v\n", err)
			// Continue without logging
		} else {
			defer chatLogger.Close()
		}
	}

	// Create orchestrator with appropriate writer
	var writer io.Writer = os.Stdout
	if chatLogger != nil {
		writer = nil // Logger will handle console output
	}

	orch := orchestrator.NewOrchestrator(orchConfig, writer)
	if chatLogger != nil {
		orch.SetLogger(chatLogger)
	}

	fmt.Println("🚀 Starting AgentPipe conversation...")
	fmt.Printf("Mode: %s | Max turns: %d | Agents: %d\n", cfg.Orchestrator.Mode, cfg.Orchestrator.MaxTurns, len(agentsList))
	if !cfg.Logging.Enabled {
		fmt.Println("📝 Chat logging disabled (use --log-dir to enable)")
	}
	fmt.Println(string(make([]byte, 60)) + "=")

	for _, a := range agentsList {
		orch.AddAgent(a)
	}

	if err := orch.Start(ctx); err != nil {
		return fmt.Errorf("orchestrator error: %w", err)
	}

	fmt.Println("\n" + string(make([]byte, 60)) + "=")
	fmt.Println("Conversation ended.")

	return nil
}
