package agent

import (
	"context"
	"fmt"
	"io"
	"time"
)

type Message struct {
	AgentID   string
	AgentName string
	Content   string
	Timestamp int64
	Role      string // "agent" or "system"
	Metrics   *ResponseMetrics
}

type ResponseMetrics struct {
	Duration     time.Duration
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Model        string
	Cost         float64
}

type AgentConfig struct {
	ID             string                 `yaml:"id"`
	Type           string                 `yaml:"type"`
	Name           string                 `yaml:"name"`
	Prompt         string                 `yaml:"prompt"`
	Announcement   string                 `yaml:"announcement"`
	Model          string                 `yaml:"model"`
	Temperature    float64                `yaml:"temperature"`
	MaxTokens      int                    `yaml:"max_tokens"`
	CustomSettings map[string]interface{} `yaml:"custom_settings"`
}

type Agent interface {
	GetID() string
	GetName() string
	GetType() string
	Initialize(config AgentConfig) error
	SendMessage(ctx context.Context, messages []Message) (string, error)
	StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error
	Announce() string
	IsAvailable() bool
	HealthCheck(ctx context.Context) error
}

type BaseAgent struct {
	ID           string
	Name         string
	Type         string
	Config       AgentConfig
	Announcement string
}

func (b *BaseAgent) GetID() string {
	return b.ID
}

func (b *BaseAgent) GetName() string {
	return b.Name
}

func (b *BaseAgent) GetType() string {
	return b.Type
}

func (b *BaseAgent) Announce() string {
	if b.Announcement != "" {
		return b.Announcement
	}
	return fmt.Sprintf("%s has joined the conversation.", b.Name)
}

func (b *BaseAgent) Initialize(config AgentConfig) error {
	b.ID = config.ID
	b.Name = config.Name
	b.Type = config.Type
	b.Config = config
	b.Announcement = config.Announcement
	return nil
}
