package main

import (
	"fmt"
	"llm-caller/cmd"
	"os"

	"github.com/joho/godotenv"
)

// Version information populated at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	BuildBy   = "unknown"
)

func main() {
	// Handle version command
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Printf("llm-caller %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Built By: %s\n", BuildBy)
		return
	}

	// Load environment variables from .env file
	_ = godotenv.Load()

	// Execute the CLI commands
	cmd.Execute()
}
