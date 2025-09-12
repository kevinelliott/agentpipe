package adapters

import (
	"fmt"
	"strings"
)

// BuildAgentPrompt creates a standard prompt for multi-agent conversations
func BuildAgentPrompt(agentName string, customPrompt string, conversation string) string {
	var prompt strings.Builder

	prompt.WriteString("You are participating in a multi-agent conversation. ")
	prompt.WriteString(fmt.Sprintf("Your name is '%s'. ", agentName))

	if customPrompt != "" {
		prompt.WriteString(customPrompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Here is the conversation so far:\n\n")
	prompt.WriteString(conversation)
	prompt.WriteString("\n\nContinue the conversation naturally as ")
	prompt.WriteString(agentName)
	prompt.WriteString(". Build on what was just said without repeating previous points. Don't announce that you're joining - just respond directly:")

	return prompt.String()
}
