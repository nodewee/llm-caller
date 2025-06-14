package cmd

import (
	"fmt"
	"os"

	"github.com/nodewee/llm-caller/pkg/config"

	"github.com/spf13/cobra"
)

var (
	cfg *config.Config
)

// Root command - simplified with clear subcommands
var rootCmd = &cobra.Command{
	Use:   "llm-caller",
	Short: "A unified CLI tool for calling various LLM services",
	Long: `LLM Caller - Call various LLM services using JSON templates

Main Commands:
  call       Execute an LLM API call using a template
  template   Manage template files (download, list, show, validate)
  config     Configure application settings
  doctor     Check configuration and environment
  version    Display detailed version information with commit hash and build time

You can also use --version flag to display basic version information.

Examples:
  llm-caller call deepseek-chat --var prompt="Hello world"
  llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json
  llm-caller config template_dir ~/my-templates
  llm-caller doctor
  llm-caller version

Use "llm-caller <command> --help" for more information about a command.`,
}

// Initialize commands and configuration
func init() {
	// Initialize config
	var err error
	cfg, err = config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Add all subcommands
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(doctorCmd)
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
