package cmd

import (
	"fmt"

	"github.com/kevinelliott/agentpipe/internal/branding"
)

// PrintLogo prints the AgentPipe ASCII art logo with sunset gradient
func PrintLogo() {
	fmt.Print("\n" + branding.ASCIILogo + "\n")
}
