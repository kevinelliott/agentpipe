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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/internal/bridge"
	"github.com/kevinelliott/agentpipe/internal/version"
	_ "github.com/kevinelliott/agentpipe/pkg/adapters"
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
	noSummary          bool
	summaryAgent       string
	jsonOutput         bool
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
	runCmd.Flags().BoolVar(&noSummary, "no-summary", false, "Disable conversation summary generation (overrides config)")
	runCmd.Flags().StringVar(&summaryAgent, "summary-agent", "", "Agent to use for summary generation (default: gemini, overrides config)")
	runCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events in JSON format (JSONL)")
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

	// Apply CLI overrides for summary
	if noSummary {
		cfg.Orchestrator.Summary.Enabled = false
	}
	if summaryAgent != "" {
		cfg.Orchestrator.Summary.Agent = summaryAgent
	}

	if err := startConversation(cobraCmd, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseAgentSpec(spec string, index int) (agent.AgentConfig, error) {
	// Parse the spec using the new model-aware parser
	agentType, model, name, err := parseAgentSpecWithModel(spec)
	if err != nil {
		return agent.AgentConfig{}, fmt.Errorf("invalid agent specification '%s': %w", spec, err)
	}

	// Auto-generate name if not provided
	if name == "" {
		name = fmt.Sprintf("%s-agent-%d", agentType, index+1)
	}

	return agent.AgentConfig{
		ID:    fmt.Sprintf("%s-%d", agentType, index),
		Type:  agentType,
		Name:  name,
		Model: model,
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

	if !jsonOutput {
		fmt.Println("🔍 Initializing agents...")
	}

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

	if !jsonOutput {
		fmt.Printf("✅ All %d agents initialized successfully\n\n", len(agentsList))
	}

	orchConfig := orchestrator.OrchestratorConfig{
		Mode:          orchestrator.ConversationMode(cfg.Orchestrator.Mode),
		TurnTimeout:   cfg.Orchestrator.TurnTimeout,
		MaxTurns:      cfg.Orchestrator.MaxTurns,
		ResponseDelay: cfg.Orchestrator.ResponseDelay,
		InitialPrompt: cfg.Orchestrator.InitialPrompt,
		Summary:       cfg.Orchestrator.Summary,
	}

	// Create logger if enabled
	var chatLogger *logger.ChatLogger
	if cfg.Logging.Enabled {
		var err error
		// Suppress console output when --json is set
		var consoleWriter io.Writer = os.Stdout
		if jsonOutput {
			consoleWriter = nil
		}
		chatLogger, err = logger.NewChatLogger(cfg.Logging.ChatLogDir, cfg.Logging.LogFormat, consoleWriter, cfg.Logging.ShowMetrics)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create chat logger: %v\n", err)
			// Continue without logging
		} else {
			defer chatLogger.Close()
		}
	}

	// Create orchestrator with appropriate writer
	var writer io.Writer = os.Stdout
	if chatLogger != nil || jsonOutput {
		writer = nil // Logger will handle console output, or suppress for JSON mode
	}

	orch := orchestrator.NewOrchestrator(orchConfig, writer)
	if chatLogger != nil {
		orch.SetLogger(chatLogger)
	}

	// Capture command information for event tracking
	commandInfo := buildCommandInfo(cmd, cfg)
	orch.SetCommandInfo(commandInfo)

	// Set up JSON stdout emitter if --json flag is set
	if jsonOutput {
		stdoutEmitter := bridge.NewStdoutEmitter(version.GetShortVersion())
		orch.SetBridgeEmitter(stdoutEmitter)

		// Set JSON emitter on logger to emit log.entry events
		if chatLogger != nil {
			chatLogger.SetJSONEmitter(stdoutEmitter)
		}

		// Reinitialize zerolog to use JSON writer for diagnostic logs
		jsonWriter := bridge.NewZerologJSONWriter(stdoutEmitter)
		level := zerolog.InfoLevel
		if verbose {
			level = zerolog.DebugLevel
		}
		log.InitLogger(jsonWriter, level, false) // false = don't use pretty console output
	} else {
		// Set up streaming bridge if enabled (only when not in JSON mode)
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
	}

	// Only show UI elements when not in JSON output mode
	if !jsonOutput {
		fmt.Println("🚀 Starting AgentPipe conversation...")
		fmt.Printf("Mode: %s | Max turns: %d | Agents: %d\n", cfg.Orchestrator.Mode, cfg.Orchestrator.MaxTurns, len(agentsList))
		if !cfg.Logging.Enabled {
			fmt.Println("📝 Chat logging disabled (use --log-dir to enable)")
		}
		fmt.Println(strings.Repeat("=", 60))
	}

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

	// Only print UI summary when not in JSON mode
	if !jsonOutput {
		fmt.Println("\n" + strings.Repeat("=", 60))
	}

	// Save conversation state if requested
	if saveState || stateFile != "" {
		if saveErr := saveConversationState(orch, cfg, time.Now()); saveErr != nil {
			log.WithError(saveErr).Error("failed to save conversation state")
			fmt.Fprintf(os.Stderr, "Warning: Failed to save conversation state: %v\n", saveErr)
		}
	}

	// Only print session summary when not in JSON output mode
	if !jsonOutput {
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
	}

	if err != nil {
		return fmt.Errorf("orchestrator error: %w", err)
	}

	return nil
}

// saveConversationState saves the current conversation state to a file.
func saveConversationState(orch *orchestrator.Orchestrator, cfg *config.Config, startedAt time.Time) error {
	messages := orch.GetMessages()
	state := conversation.NewState(messages, cfg, startedAt)

	// Populate summary fields if available
	if summary := orch.GetSummary(); summary != nil {
		state.Metadata.ShortText = summary.ShortText
		state.Metadata.Text = summary.Text
	}

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

// buildCommandInfo constructs a CommandInfo struct from the cobra command and config
func buildCommandInfo(cmd *cobra.Command, cfg *config.Config) *bridge.CommandInfo {
	// Build the full command string
	args := os.Args
	fullCommand := strings.Join(args, " ")

	// Build options map with all relevant flags
	options := make(map[string]string)

	// Add all flags that were explicitly set
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		options[flag.Name] = flag.Value.String()
	})

	// Build agent list string for readability
	agentList := make([]string, 0, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		agentSpec := fmt.Sprintf("%s:%s", agent.Type, agent.Name)
		agentList = append(agentList, agentSpec)
	}
	if len(agentList) > 0 {
		options["agents_list"] = strings.Join(agentList, ",")
	}

	return &bridge.CommandInfo{
		FullCommand:    fullCommand,
		Args:           args[1:], // Exclude program name
		Mode:           cfg.Orchestrator.Mode,
		MaxTurns:       cfg.Orchestrator.MaxTurns,
		InitialPrompt:  cfg.Orchestrator.InitialPrompt,
		ConfigFile:     configPath,
		TUIEnabled:     useTUI,
		LoggingEnabled: cfg.Logging.Enabled,
		ShowMetrics:    showMetrics,
		Timeout:        int(cfg.Orchestrator.TurnTimeout.Seconds()),
		Options:        options,
	}
}
