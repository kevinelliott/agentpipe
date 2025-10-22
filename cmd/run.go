package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/kevinelliott/agentpipe/pkg/adapters"
	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/internal/version"
	"github.com/kevinelliott/agentpipe/pkg/agent"
	"github.com/kevinelliott/agentpipe/pkg/config"
	"github.com/kevinelliott/agentpipe/pkg/conversation"
	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/logger"
	"github.com/kevinelliott/agentpipe/pkg/orchestrator"
	"github.com/kevinelliott/agentpipe/pkg/tui"
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
	watchConfig        bool
	saveState          bool
	stateFile          string
	streamEnabled      bool
	noStream           bool
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
	runCmd.Flags().IntVar(&healthCheckTimeout, "health-check-timeout", 5, "Health check timeout in seconds")
	runCmd.Flags().StringVar(&chatLogDir, "log-dir", "", "Directory to save chat logs (default: ~/.agentpipe/chats)")
	runCmd.Flags().BoolVar(&disableLogging, "no-log", false, "Disable chat logging")
	runCmd.Flags().BoolVar(&showMetrics, "metrics", false, "Show response metrics (duration, tokens, cost)")
	runCmd.Flags().BoolVar(&watchConfig, "watch-config", false, "Watch config file for changes and hot-reload (requires --config)")
	runCmd.Flags().BoolVar(&saveState, "save-state", false, "Save conversation state on exit (to ~/.agentpipe/states)")
	runCmd.Flags().StringVar(&stateFile, "state-file", "", "Specific file path to save conversation state")
	runCmd.Flags().BoolVar(&streamEnabled, "stream", false, "Enable streaming to AgentPipe Web for this run (overrides config)")
	runCmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming to AgentPipe Web for this run (overrides config)")
}

func runConversation(cobraCmd *cobra.Command, args []string) {
	var cfg *config.Config
	var err error

	if configPath != "" {
		log.WithField("config_path", configPath).Debug("loading configuration from file")
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			log.WithError(err).WithField("config_path", configPath).Error("failed to load configuration")
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		log.WithFields(map[string]interface{}{
			"config_path": configPath,
			"agents":      len(cfg.Agents),
			"mode":        cfg.Orchestrator.Mode,
		}).Info("configuration loaded successfully")
	} else if len(agents) > 0 {
		log.WithField("agent_count", len(agents)).Debug("creating configuration from CLI arguments")
		cfg = config.NewDefaultConfig()
		for i, agentSpec := range agents {
			agentCfg, err := parseAgentSpec(agentSpec, i)
			if err != nil {
				log.WithError(err).WithField("agent_spec", agentSpec).Error("failed to parse agent specification")
				fmt.Fprintf(os.Stderr, "Error parsing agent spec: %v\n", err)
				os.Exit(1)
			}
			cfg.Agents = append(cfg.Agents, agentCfg)
		}
	} else {
		log.Error("no configuration source specified (need --config or --agents)")
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up config watcher if requested
	var configWatcher *config.ConfigWatcher
	if watchConfig && configPath != "" {
		var err error
		configWatcher, err = config.NewConfigWatcher(configPath)
		if err != nil {
			log.WithError(err).Error("failed to create config watcher")
			fmt.Fprintf(os.Stderr, "Warning: Failed to create config watcher: %v\n", err)
		} else {
			// Register callback to log config changes
			configWatcher.OnConfigChange(func(oldConfig, newConfig *config.Config) {
				log.WithFields(map[string]interface{}{
					"old_agents":    len(oldConfig.Agents),
					"new_agents":    len(newConfig.Agents),
					"old_max_turns": oldConfig.Orchestrator.MaxTurns,
					"new_max_turns": newConfig.Orchestrator.MaxTurns,
					"old_mode":      oldConfig.Orchestrator.Mode,
					"new_mode":      newConfig.Orchestrator.Mode,
				}).Info("configuration file changed")

				fmt.Println("\n📝 Configuration file changed!")
				fmt.Printf("   Mode: %s → %s\n", oldConfig.Orchestrator.Mode, newConfig.Orchestrator.Mode)
				fmt.Printf("   Max Turns: %d → %d\n", oldConfig.Orchestrator.MaxTurns, newConfig.Orchestrator.MaxTurns)
				fmt.Printf("   Agents: %d → %d\n", len(oldConfig.Agents), len(newConfig.Agents))
				fmt.Println("   Note: Some changes require restarting the conversation")
			})

			// Start watching in background
			go configWatcher.StartWatching()
			defer configWatcher.StopWatching()

			fmt.Println("👀 Config file watching enabled (changes will be detected automatically)")
		}
	}

	// Track graceful shutdown for summary display
	gracefulShutdown := false
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n⏸️  Interrupted. Shutting down gracefully...")
		gracefulShutdown = true
		cancel()
	}()

	if useTUI {
		// Use enhanced TUI - agent initialization will happen inside TUI
		skipHealthCheck, err := cmd.Flags().GetBool("skip-health-check")
		if err != nil {
			skipHealthCheck = false
		}
		return tui.RunEnhanced(ctx, cfg, nil, skipHealthCheck, healthCheckTimeout, configPath)
	}

	// Non-TUI mode: initialize agents here
	agentsList := make([]agent.Agent, 0)

	verbose := viper.GetBool("verbose")

	fmt.Println("🔍 Initializing agents...")

	for _, agentCfg := range cfg.Agents {
		if verbose {
			fmt.Printf("  Creating agent %s (type: %s)...\n", agentCfg.Name, agentCfg.Type)
		}

		log.WithFields(map[string]interface{}{
			"agent_name": agentCfg.Name,
			"agent_type": agentCfg.Type,
			"agent_id":   agentCfg.ID,
		}).Debug("creating agent")

		a, err := agent.CreateAgent(agentCfg)
		if err != nil {
			log.WithError(err).WithFields(map[string]interface{}{
				"agent_name": agentCfg.Name,
				"agent_type": agentCfg.Type,
			}).Error("failed to create agent")
			return fmt.Errorf("failed to create agent %s: %w", agentCfg.Name, err)
		}

		if !a.IsAvailable() {
			log.WithFields(map[string]interface{}{
				"agent_name": agentCfg.Name,
				"agent_type": agentCfg.Type,
			}).Error("agent CLI not available")
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
				timeout = 5 * time.Second
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

	// Set up streaming bridge if enabled
	shouldStream := determineShouldStream(streamEnabled, noStream)
	if shouldStream {
		bridgeConfig := bridge.LoadConfig()
		if bridgeConfig.Enabled || streamEnabled {
			// Override config enabled setting if --stream was specified
			if streamEnabled {
				bridgeConfig.Enabled = true
			}

			emitter := bridge.NewEmitter(bridgeConfig, version.GetShortVersion())
			orch.SetBridgeEmitter(emitter)

			if verbose {
				fmt.Printf("🌐 Streaming enabled (conversation ID: %s)\n", emitter.GetConversationID())
			}
		}
	}

	fmt.Println("🚀 Starting AgentPipe conversation...")
	fmt.Printf("Mode: %s | Max turns: %d | Agents: %d\n", cfg.Orchestrator.Mode, cfg.Orchestrator.MaxTurns, len(agentsList))
	if !cfg.Logging.Enabled {
		fmt.Println("📝 Chat logging disabled (use --log-dir to enable)")
	}
	fmt.Println(strings.Repeat("=", 60))

	log.WithFields(map[string]interface{}{
		"mode":         cfg.Orchestrator.Mode,
		"max_turns":    cfg.Orchestrator.MaxTurns,
		"agent_count":  len(agentsList),
		"logging":      cfg.Logging.Enabled,
		"show_metrics": cfg.Logging.ShowMetrics,
	}).Info("starting agentpipe conversation")

	for _, a := range agentsList {
		orch.AddAgent(a)
	}

	err := orch.Start(ctx)

	if err != nil {
		log.WithError(err).Error("orchestrator error during conversation")
	} else {
		log.Info("conversation completed successfully")
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))

	// Save conversation state if requested
	if saveState || stateFile != "" {
		if saveErr := saveConversationState(orch, cfg, time.Now()); saveErr != nil {
			log.WithError(saveErr).Error("failed to save conversation state")
			fmt.Fprintf(os.Stderr, "Warning: Failed to save conversation state: %v\n", saveErr)
		}
	}

	// Always print session summary (whether interrupted or completed normally)
	if gracefulShutdown {
		fmt.Println("📊 Session Summary (Interrupted)")
	} else if err != nil {
		fmt.Println("📊 Session Summary (Ended with Error)")
	} else {
		fmt.Println("📊 Session Summary (Completed)")
	}
	fmt.Println(strings.Repeat("=", 60))
	printSessionSummary(orch, cfg)

	if err != nil {
		return fmt.Errorf("orchestrator error: %w", err)
	}

	return nil
}

// saveConversationState saves the current conversation state to a file.
func saveConversationState(orch *orchestrator.Orchestrator, cfg *config.Config, startedAt time.Time) error {
	messages := orch.GetMessages()
	state := conversation.NewState(messages, cfg, startedAt)

	// Determine save path
	var savePath string
	if stateFile != "" {
		savePath = stateFile
	} else {
		// Use default state directory
		stateDir, err := conversation.GetDefaultStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		savePath = filepath.Join(stateDir, conversation.GenerateStateFileName())
	}

	// Save state
	if err := state.Save(savePath); err != nil {
		return err
	}

	fmt.Printf("\n💾 Conversation state saved to: %s\n", savePath)
	log.WithFields(map[string]interface{}{
		"path":     savePath,
		"messages": len(messages),
	}).Info("conversation state saved successfully")

	return nil
}

// printSessionSummary prints a summary of the conversation session
func printSessionSummary(orch *orchestrator.Orchestrator, cfg *config.Config) {
	messages := orch.GetMessages()

	// Calculate statistics
	totalMessages := 0
	agentMessages := 0
	systemMessages := 0
	totalCost := 0.0
	totalTime := time.Duration(0)
	totalTokens := 0

	for _, msg := range messages {
		totalMessages++

		if msg.Role == "agent" {
			agentMessages++
			if msg.Metrics != nil {
				if msg.Metrics.Cost > 0 {
					totalCost += msg.Metrics.Cost
				}
				if msg.Metrics.Duration > 0 {
					totalTime += msg.Metrics.Duration
				}
				if msg.Metrics.TotalTokens > 0 {
					totalTokens += msg.Metrics.TotalTokens
				}
			}
		} else if msg.Role == "system" {
			systemMessages++
		}
	}

	// Display summary
	fmt.Printf("Total Messages:      %d\n", totalMessages)
	fmt.Printf("  Agent Messages:    %d\n", agentMessages)
	fmt.Printf("  System Messages:   %d\n", systemMessages)

	if totalTokens > 0 {
		fmt.Printf("Total Tokens:        %d\n", totalTokens)
	}

	// Format time
	if totalTime > 0 {
		if totalTime < time.Second {
			fmt.Printf("Total Time:          %dms\n", totalTime.Milliseconds())
		} else if totalTime < time.Minute {
			fmt.Printf("Total Time:          %.1fs\n", totalTime.Seconds())
		} else {
			minutes := int(totalTime.Minutes())
			seconds := int(totalTime.Seconds()) % 60
			fmt.Printf("Total Time:          %dm%ds\n", minutes, seconds)
		}
	}

	if totalCost > 0 {
		fmt.Printf("Total Cost:          $%.4f\n", totalCost)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Session ended. All messages logged.")
}

// determineShouldStream determines if streaming should be enabled based on CLI flags.
// Priority: --no-stream > --stream > config file setting
func determineShouldStream(streamEnabled, noStream bool) bool {
	// If both flags are set, --no-stream takes priority
	if streamEnabled && noStream {
		return false
	}

	// If --no-stream is set, disable streaming
	if noStream {
		return false
	}

	// If --stream is set, enable streaming
	if streamEnabled {
		return true
	}

	// Otherwise, use config file setting (checked later)
	// We return true here to let the config be checked
	bridgeConfig := bridge.LoadConfig()
	return bridgeConfig.Enabled
}
