// +build ignore

// This tool fetches provider configs from Catwalk and generates providers.json
// Run with: go run scripts/update-providers.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kevinelliott/agentpipe/internal/providers"
)

func main() {
	fmt.Println("Fetching provider configs from Catwalk...")

	config, err := providers.FetchProvidersFromCatwalk()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching providers: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fetched %d providers\n", len(config.Providers))

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to internal/providers/providers.json
	outputPath := filepath.Join("internal", "providers", "providers.json")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote %s\n", outputPath)
	fmt.Println("Updated at:", config.UpdatedAt)
	fmt.Println("Source:", config.Source)
}
