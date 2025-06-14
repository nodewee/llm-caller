package cmd

import (
	"fmt"
	"strings"

	"github.com/nodewee/llm-caller/pkg/config"

	"github.com/spf13/cobra"
)

// Config command
var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Configure application settings",
	Long: `Manage application configuration including template directory and API keys file.

Configuration is stored in ~/.llm-caller/config.yaml

Usage:
  config [key]            Get the value for a specific key
  config [key] [value]    Set a value for a specific key
  config ls               List all configuration values
  config rm [key]         Remove a specific key (revert to default)

Available settings:
  template_dir - Directory where template files are stored
  secret_file  - Path to JSON file containing API keys
  
Examples:
  llm-caller config template_dir               # Get value
  llm-caller config template_dir ~/my-templates # Set value
  llm-caller config ls                         # List all settings
  llm-caller config rm template_dir           # Remove setting (revert to default)`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

// Config subcommands
var configLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all configuration values",
	Long:  `Display all current configuration values including file location.`,
	Args:  cobra.NoArgs,
	RunE:  runConfigList,
}

var configRmCmd = &cobra.Command{
	Use:   "rm <key>",
	Short: "Remove a configuration value",
	Long: `Remove a configuration value, reverting to default.

Examples:
  llm-caller config rm template_dir
  llm-caller config rm secret_file`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigRm,
}

func init() {
	// Config subcommands
	configCmd.AddCommand(configLsCmd)
	configCmd.AddCommand(configRmCmd)
}

// Config command handler - unified get/set functionality
func runConfig(cmd *cobra.Command, args []string) error {
	// If no arguments, show usage
	if len(args) == 0 {
		return cmd.Help()
	}

	key := args[0]

	// If one argument, get the value (former get command)
	if len(args) == 1 {
		value := cfg.Get(key)
		if value == nil {
			return fmt.Errorf("key %s not found", key)
		}
		fmt.Println(value)
		return nil
	}

	// If two arguments, set the value (former set command)
	value := args[1]

	// Validate key
	validKeys := []string{config.KeyTemplateDir, config.KeySecretFile}
	validKey := false
	for _, vk := range validKeys {
		if key == vk {
			validKey = true
			break
		}
	}
	if !validKey {
		return fmt.Errorf("invalid key: %s, valid keys are: %s", key, strings.Join(validKeys, ", "))
	}

	if err := cfg.Set(key, value); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fmt.Printf("Set %s to %s\n", key, value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	configPath := cfg.GetConfigFilePath()
	fmt.Printf("Configuration file: %s\n\n", configPath)

	settings := cfg.List()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
	return nil
}

func runConfigRm(cmd *cobra.Command, args []string) error {
	key := args[0]

	err := cfg.Delete(key)
	if err != nil {
		if strings.Contains(err.Error(), "not found in configuration") {
			fmt.Printf("Key %s not found (already unset or never existed)\n", key)
			return nil
		}
		return fmt.Errorf("failed to remove config: %w", err)
	}

	fmt.Printf("Removed %s (reverted to default)\n", key)
	return nil
}
