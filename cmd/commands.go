package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"llm-caller/pkg/config"
	"llm-caller/pkg/download"
	"llm-caller/pkg/llm"
	"llm-caller/pkg/templates"
	"llm-caller/pkg/utils"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfg          *config.Config
	templateFlag string
	varFlags     []string
	apiKeyFlag   string
	outputFlag   string
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "llm-caller",
	Short: "A unified CLI tool for calling various LLM services via HTTP requests",
	Long: `A CLI tool for calling LLM services using JSON templates.

Usage:
  llm-caller -t <template> --var name:type:value [--api-key <key>] [-o <output>]

Variable Types:
  text   - Use value directly
  file   - Read content from file
  base64 - Decode base64 content

Examples:
  llm-caller -t deepseek-chat --var prompt:text:"Hello world"
  llm-caller -t translate --var text:file:doc.txt -o result.txt`,
	RunE: runRoot,
}

// Config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration settings",
	Long: `Manage application configuration settings including template directory and API keys file.

Configuration is stored in ~/.llm-caller/config.yaml and includes:

• template_dir - Directory where template files are stored (default: ~/.llm-caller/templates)
• secret_file  - Path to JSON file containing API keys (default: ~/.llm-caller/secret file(json))

The configuration system supports:
- Cross-platform path handling
- Automatic directory creation
- Fallback to default values
- Complete CRUD operations (create, read, update, delete)

Available subcommands:
  get    - Get a specific configuration value
  set    - Set a configuration value  
  list   - List all configuration values and show config file location
  delete - Delete a configuration value from user settings`,
}

// Config get command
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value by key",
	Long: `Get a configuration value by its key. Returns the current value including defaults.

Available configuration keys:
  template_dir - Directory where template files are stored
  secret_file  - Path to JSON file containing API keys

Examples:
  llm-caller config get template_dir
  llm-caller config get secret_file

Note: This command returns the effective value (user setting or default).`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

// Config set command
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value by specifying a key and value. The value will be saved 
to the configuration file and persist across sessions.

Available configuration keys:
  template_dir - Directory where template files are stored
                 Supports both absolute and relative paths
                 Directory will be created if it doesn't exist
  
  secret_file  - Path to JSON file containing API keys
                 File should contain provider-specific keys like:
                 {"deepseek_api_key": "sk-xxx", "openai_api_key": "sk-xxx"}

Examples:
  llm-caller config set template_dir ~/my-templates
  llm-caller config set template_dir /absolute/path/to/templates
  llm-caller config set secret_file ~/.llm-caller/api-keys.json

Note: Only valid configuration keys are accepted. Invalid keys will be rejected.`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

// Config list command
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long: `Display all current configuration values including both user settings and defaults.
	
This command shows:
• The configuration file location
• All configuration keys and their current effective values
• Both user-defined settings and system defaults

The output includes the full path to the configuration file, making it easy to
locate and manually edit if needed.

Example output:
  Configuration file: /Users/username/.llm-caller/config.yaml
  
  template_dir: /Users/username/.llm-caller/templates
  secret_file: /Users/username/.llm-caller/secret file(json)`,
	RunE: runConfigList,
}

// Config delete command
var configDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a configuration value",
	Long: `Delete a configuration value from the user configuration file. This removes 
the key from your personal settings, causing the system to fall back to default values.

Important notes:
• Only user-defined configuration values can be deleted
• After deletion, the system will use the default value for that key
• If the key doesn't exist in user configuration, the operation succeeds silently
• System defaults cannot be deleted, only user overrides

Common configuration keys:
  template_dir - Directory where template files are stored
  secret_file  - Path to JSON file containing API keys

Examples:
  llm-caller config delete template_dir
  llm-caller config delete secret_file
  
  # Delete any legacy or custom keys
  llm-caller config delete old_setting

After deletion, you can verify the change with:
  llm-caller config list`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigDelete,
}

// Template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage template files",
	Long: `Manage template files including downloading from GitHub repositories and listing available templates.

Templates are JSON files that define how to call LLM services. The system uses two separate directories:

• Downloaded templates: ~/.llm-caller/templates (managed by download command)
• User custom templates: Configured template_dir (managed by user)

This separation ensures downloaded templates don't interfere with user-created templates.

Available subcommands:
  download - Download a template from GitHub URL
  list     - List available templates from all directories`,
}

// Template download command
var templateDownloadCmd = &cobra.Command{
	Use:   "download <github-url>",
	Short: "Download a template from GitHub",
	Long: `Download a template file from a GitHub repository URL and save it to the default templates directory.

The command accepts GitHub blob URLs and automatically converts them to raw download URLs.
Downloaded templates are always saved to ~/.llm-caller/templates (separate from user custom templates).

Supported URL formats:
  https://github.com/owner/repo/blob/branch/filename.json
  https://github.com/nodewee/llm-calling-templates/blob/main/qwen-vl-ocr-image.json

Examples:
  llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/qwen-vl-ocr-image.json
  llm-caller template download https://github.com/owner/repo/blob/main/custom-template.json

The downloaded template will be:
• Validated for basic JSON format
• Saved to ~/.llm-caller/templates with the original filename
• Available for use with the -t flag (checked after user custom templates)

After downloading, you can use the template with:
  llm-caller -t filename --var key:type:value`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateDownload,
}

// Template list command
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates from all directories",
	Long: `List all available template files from both user configured directory and 
the default app config directory.

This command shows:
• User configured template directory (if set)
• Default app config template directory (~/.llm-caller/templates)
• All .json template files in each directory
• Total count of templates found

The templates are listed in priority order (user templates are checked first).

Examples:
  llm-caller template list
  
Output format:
  User templates (/custom/path/to/templates):
    - my-custom-template.json
    - another-template.json
  
  Downloaded templates (~/.llm-caller/templates):
    - qwen-vl-ocr-image.json
    - deepseek-chat.json
  
  Total: 4 templates found`,
	RunE: runTemplateList,
}

// Initialize commands
func init() {
	// Initialize config
	var err error
	cfg, err = config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Root command flags
	rootCmd.Flags().StringVarP(&templateFlag, "template", "t", "", "(Required) Template file name or path. Supports relative/absolute paths with automatic .json extension")
	rootCmd.Flags().StringArrayVar(&varFlags, "var", []string{}, "(Optional) Variable replacement in format name:type:value. Types: text|file|base64. Can be used multiple times")
	rootCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "(Optional) API key to override configured sources (secret file or environment variables)")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "(Optional) File path to save the result. If not specified, results are printed to stdout.")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configDeleteCmd)

	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateDownloadCmd)
	templateCmd.AddCommand(templateListCmd)
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// runRoot handles the root command
func runRoot(cmd *cobra.Command, args []string) error {
	// Check required template flag
	if templateFlag == "" {
		return fmt.Errorf("template flag (-t) is required")
	}

	// Parse var flags
	replaceVars, err := parseVarFlags(varFlags)
	if err != nil {
		return fmt.Errorf("failed to parse var flags: %w", err)
	}

	// Load the template
	template, err := templates.LoadTemplate(cfg, templateFlag)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Get API key based on priority:
	// 1. Command-line flag (--api-key)
	// 2. API keys file (secret_file)
	// 3. Environment variable
	apiKey, err := getAPIKey(apiKeyFlag, cfg, template)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Add api_key to replacement variables
	if replaceVars == nil {
		replaceVars = make(map[string]string)
	}
	replaceVars["api_key"] = apiKey

	// Replace variables if needed
	if len(replaceVars) > 0 {
		template.ReplaceVariables(replaceVars)
	}

	// Get the provider
	provider, err := llm.GetProvider(template, apiKey)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Call the provider
	result, err := provider.Call(template)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Print result to stdout or file
	if outputFlag == "" {
		fmt.Print(result)
	} else {
		err = os.WriteFile(outputFlag, []byte(result), utils.GetFilePermissions())
		if err != nil {
			return fmt.Errorf("failed to write output to file: %w", err)
		}
		fmt.Printf("Result saved to %s\n", outputFlag)
	}
	return nil
}

// parseVarFlags parses --var flags in format name:type:value
func parseVarFlags(varFlags []string) (map[string]string, error) {
	replaceVars := make(map[string]string)

	for _, varFlag := range varFlags {
		parts := strings.SplitN(varFlag, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid var format, expected name:type:value: %s", varFlag)
		}

		name := parts[0]
		varType := parts[1]
		value := parts[2]

		if name == "" {
			return nil, fmt.Errorf("variable name cannot be empty in: %s", varFlag)
		}

		switch varType {
		case "text":
			// Use value as-is
			replaceVars[name] = value

		case "file":
			// Read value from file
			if value == "" {
				return nil, fmt.Errorf("file path cannot be empty for variable %s", name)
			}
			fileContent, err := os.ReadFile(value)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s for variable %s: %w", value, name, err)
			}
			replaceVars[name] = string(fileContent)

		case "base64":
			// Decode base64 value
			if value == "" {
				return nil, fmt.Errorf("base64 value cannot be empty for variable %s", name)
			}
			decoded, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 value for variable %s: %w", name, err)
			}
			replaceVars[name] = string(decoded)

		default:
			return nil, fmt.Errorf("unsupported variable type '%s' for variable %s, supported types: text, file, base64", varType, name)
		}
	}

	return replaceVars, nil
}

// getAPIKey retrieves API key based on priority: CLI > file > environment
func getAPIKey(cliAPIKey string, cfg *config.Config, template *templates.Template) (string, error) {
	// 1. CLI argument has highest priority
	if cliAPIKey != "" {
		return cliAPIKey, nil
	}

	// 2. Try to load from secret file
	apiKeysFile := cfg.GetString(config.KeySecretFile)
	if apiKeysFile != "" {
		if keys, err := loadApiKeys(apiKeysFile); err == nil {
			// Try provider-specific key first
			if template.Provider != "" {
				if key, ok := keys[template.Provider+"_api_key"]; ok && key != "" {
					return key, nil
				}
			}
			// Try generic keys
			for _, keyName := range []string{"api_key", "default_api_key"} {
				if key, ok := keys[keyName]; ok && key != "" {
					return key, nil
				}
			}
		}
	}

	// 3. Try environment variables
	envKeys := []string{"API_KEY"}
	if template.Provider != "" {
		envKeys = append([]string{strings.ToUpper(template.Provider) + "_API_KEY"}, envKeys...)
	}

	for _, envKey := range envKeys {
		if envValue := utils.GetEnvironmentVariableCaseInsensitive(envKey); envValue != "" {
			return envValue, nil
		}
	}

	return "", fmt.Errorf("API key not found. Please provide it via --api-key flag, secret file, or %s environment variable",
		strings.ToUpper(template.Provider)+"_API_KEY")
}

// loadApiKeys loads API keys from a JSON file
func loadApiKeys(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var keys map[string]string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// runConfigGet handles the config get command
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := cfg.Get(key)
	if value == nil {
		return fmt.Errorf("key %s not found", key)
	}
	fmt.Println(value)
	return nil
}

// runConfigSet handles the config set command
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Ensure the key is valid
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

	// Set the value
	if err := cfg.Set(key, value); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fmt.Printf("Set %s to %s\n", key, value)
	return nil
}

// runConfigList handles the config list command
func runConfigList(cmd *cobra.Command, args []string) error {
	// Show config file path
	configPath := cfg.GetConfigFilePath()
	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Println()

	settings := cfg.List()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
	return nil
}

// runConfigDelete handles the config delete command
func runConfigDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Delete the value (no key validation - allow any key)
	err := cfg.Delete(key)
	if err != nil {
		// If the key doesn't exist, treat it as successful deletion
		if strings.Contains(err.Error(), "not found in configuration") {
			fmt.Printf("Key %s not found (already deleted or never existed)\n", key)
			return nil
		}
		return fmt.Errorf("failed to delete config: %w", err)
	}

	fmt.Printf("Deleted %s\n", key)
	return nil
}

// runTemplateDownload handles the template download command
func runTemplateDownload(cmd *cobra.Command, args []string) error {
	githubURL := args[0]

	// Always download to the default app config templates directory
	defaultTemplateDir, err := config.GetDefaultTemplateDir()
	if err != nil {
		return fmt.Errorf("failed to get default template directory: %w", err)
	}

	// Ensure the default template directory exists
	if err := utils.CreateDirWithPlatformPermissions(defaultTemplateDir); err != nil {
		return fmt.Errorf("failed to create default template directory: %w", err)
	}

	// Create downloader and download the template
	downloader := download.NewGitHubDownloader()
	filePath, err := downloader.DownloadTemplate(githubURL, defaultTemplateDir)
	if err != nil {
		return fmt.Errorf("failed to download template: %w", err)
	}

	// Validate the downloaded template
	if err := downloader.ValidateTemplateFile(filePath); err != nil {
		// Remove the invalid file
		os.Remove(filePath)
		return fmt.Errorf("downloaded file is not a valid template: %w", err)
	}

	fmt.Printf("Template successfully downloaded and saved to: %s\n", filePath)
	return nil
}

// runTemplateList handles the template list command
func runTemplateList(cmd *cobra.Command, args []string) error {
	var totalCount int

	// Get directories
	userTemplateDir := cfg.GetString(config.KeyTemplateDir)
	defaultTemplateDir, err := config.GetDefaultTemplateDir()
	if err != nil {
		return fmt.Errorf("failed to get default template directory: %w", err)
	}

	// List templates from user configured directory
	if userTemplateDir != "" {
		userTemplates, err := templates.ListTemplates(userTemplateDir)
		if err != nil {
			return fmt.Errorf("failed to list user templates: %w", err)
		}

		fmt.Printf("User templates (%s):\n", userTemplateDir)
		if len(userTemplates) == 0 {
			fmt.Println("  (no templates found)")
		} else {
			for _, template := range userTemplates {
				fmt.Printf("  - %s\n", template)
			}
		}
		totalCount += len(userTemplates)
		fmt.Println()
	}

	// List templates from default app config directory
	defaultTemplates, err := templates.ListTemplates(defaultTemplateDir)
	if err != nil {
		return fmt.Errorf("failed to list default templates: %w", err)
	}

	fmt.Printf("Downloaded templates (%s):\n", defaultTemplateDir)
	if len(defaultTemplates) == 0 {
		fmt.Println("  (no templates found)")
	} else {
		for _, template := range defaultTemplates {
			fmt.Printf("  - %s\n", template)
		}
	}
	totalCount += len(defaultTemplates)

	fmt.Printf("\nTotal: %d templates found\n", totalCount)
	return nil
}
