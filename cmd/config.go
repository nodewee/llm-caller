package cmd

import (
	"fmt"
	"strings"

	"github.com/nodewee/llm-caller/pkg/config"

	"github.com/spf13/cobra"
)

// Config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure application settings",
	Long: `Manage application configuration including template directory and API keys file.

Configuration is stored in ~/.llm-caller/config.yaml

Available settings:
  template_dir - Directory where template files are stored
  secret_file  - Path to JSON file containing API keys`,
}

// Config subcommands
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Valid keys: template_dir, secret_file

Examples:
  llm-caller config set template_dir ~/my-templates
  llm-caller config set secret_file ~/.api-keys.json`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a configuration value by its key.

Examples:
  llm-caller config get template_dir
  llm-caller config get secret_file`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long:  `Display all current configuration values including file location.`,
	RunE:  runConfigList,
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration value",
	Long: `Remove a configuration value, reverting to default.

Examples:
  llm-caller config unset template_dir
  llm-caller config unset secret_file`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigUnset,
}

func init() {
	// Config subcommands
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUnsetCmd)
}

// Config command handlers
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
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

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := cfg.Get(key)
	if value == nil {
		return fmt.Errorf("key %s not found", key)
	}
	fmt.Println(value)
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

func runConfigUnset(cmd *cobra.Command, args []string) error {
	key := args[0]

	err := cfg.Delete(key)
	if err != nil {
		if strings.Contains(err.Error(), "not found in configuration") {
			fmt.Printf("Key %s not found (already unset or never existed)\n", key)
			return nil
		}
		return fmt.Errorf("failed to unset config: %w", err)
	}

	fmt.Printf("Unset %s (reverted to default)\n", key)
	return nil
}
