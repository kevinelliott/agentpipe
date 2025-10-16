// Package config provides configuration management for AgentPipe.
package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/pkg/log"
)

// ConfigChangeCallback is called when the configuration file changes.
// It receives the old and new configurations.
type ConfigChangeCallback func(oldConfig, newConfig *Config)

// ConfigWatcher watches a configuration file for changes and reloads it.
type ConfigWatcher struct {
	mu              sync.RWMutex
	config          *Config
	configPath      string
	viper           *viper.Viper
	callbacks       []ConfigChangeCallback
	stopChan        chan struct{}
	reloadInProcess bool
}

// NewConfigWatcher creates a new configuration watcher.
// It loads the initial configuration and sets up file watching.
func NewConfigWatcher(configPath string) (*ConfigWatcher, error) {
	// Load initial config
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	// Read initial config with viper
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config with viper: %w", err)
	}

	watcher := &ConfigWatcher{
		config:     config,
		configPath: configPath,
		viper:      v,
		callbacks:  make([]ConfigChangeCallback, 0),
		stopChan:   make(chan struct{}),
	}

	log.WithField("config_path", configPath).Info("config watcher initialized")

	return watcher, nil
}

// GetConfig returns the current configuration (thread-safe).
func (cw *ConfigWatcher) GetConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.config
}

// OnConfigChange registers a callback to be invoked when the config changes.
// Callbacks are executed in the order they were registered.
func (cw *ConfigWatcher) OnConfigChange(callback ConfigChangeCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// StartWatching begins monitoring the configuration file for changes.
// When a change is detected, the config is reloaded and callbacks are invoked.
// This method blocks, so it should typically be run in a goroutine.
func (cw *ConfigWatcher) StartWatching() {
	cw.viper.OnConfigChange(func(e fsnotify.Event) {
		cw.handleConfigChange(e)
	})

	cw.viper.WatchConfig()

	log.WithField("config_path", cw.configPath).Info("started watching config file for changes")

	// Block until stopped
	<-cw.stopChan
}

// StopWatching stops monitoring the configuration file.
func (cw *ConfigWatcher) StopWatching() {
	close(cw.stopChan)
	log.Info("stopped watching config file")
}

// handleConfigChange is called when the config file changes.
func (cw *ConfigWatcher) handleConfigChange(e fsnotify.Event) {
	cw.mu.Lock()
	if cw.reloadInProcess {
		cw.mu.Unlock()
		return
	}
	cw.reloadInProcess = true
	cw.mu.Unlock()

	defer func() {
		cw.mu.Lock()
		cw.reloadInProcess = false
		cw.mu.Unlock()
	}()

	log.WithFields(map[string]interface{}{
		"event":       e.Op.String(),
		"config_path": e.Name,
	}).Info("config file change detected")

	// Reload the config
	newConfig, err := LoadConfig(cw.configPath)
	if err != nil {
		log.WithError(err).WithField("config_path", cw.configPath).Error("failed to reload config")
		return
	}

	cw.mu.Lock()
	oldConfig := cw.config
	cw.config = newConfig
	callbacks := cw.callbacks
	cw.mu.Unlock()

	log.WithFields(map[string]interface{}{
		"config_path": cw.configPath,
		"agents":      len(newConfig.Agents),
		"mode":        newConfig.Orchestrator.Mode,
		"max_turns":   newConfig.Orchestrator.MaxTurns,
	}).Info("config reloaded successfully")

	// Invoke callbacks
	for _, callback := range callbacks {
		go func(cb ConfigChangeCallback) {
			defer func() {
				if r := recover(); r != nil {
					log.WithField("panic", r).Error("config change callback panicked")
				}
			}()
			cb(oldConfig, newConfig)
		}(callback)
	}
}
