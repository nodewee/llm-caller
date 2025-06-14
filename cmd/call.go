package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nodewee/llm-caller/pkg/config"
	"github.com/nodewee/llm-caller/pkg/llm"
	"github.com/nodewee/llm-caller/pkg/templates"
	"github.com/nodewee/llm-caller/pkg/utils"
	"github.com/spf13/cobra"
)

// Call command flags
var (
	varFlags   []string
	apiKeyFlag string
	outputFlag string
)

// Call command - main functionality
var callCmd = &cobra.Command{
	Use:   "call <template>",
	Short: "Execute an LLM API call using a template",
	Long: `Execute an LLM API call using a specified template with variable substitution.

This command loads a template, replaces variables, and makes an HTTP request to the LLM service.
The result is either printed to stdout or saved to a file.

API keys are checked in this order:
1. --api-key command line flag
2. Keys file (configured with 'config secret_file')
3. Environment variables (provider-specific keys checked first)

API keys are optional for local LLMs like Ollama that don't require authentication.

Examples:
  llm-caller call deepseek-chat --var prompt="Hello world"
  llm-caller call translate --var text:file:doc.txt -o result.txt
  llm-caller call gpt-4 --var prompt="Explain AI" --api-key sk-xxx
  llm-caller call ollama-local --var prompt="Tell me a joke" # API key is optional`,
	Args: cobra.ExactArgs(1),
	RunE: runCall,
}

func init() {
	// Call command flags
	callCmd.Flags().StringArrayVar(&varFlags, "var", []string{}, "Variable in format name:value or name:type:value (text|file|base64)")
	callCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key (optional, overrides config and environment)")
	callCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path (default: stdout)")
}

// runCall handles the call command
func runCall(cmd *cobra.Command, args []string) error {
	templateFlag := args[0] // Get template from first argument

	// Parse var flags with improved format support
	replaceVars, err := parseVarFlags(varFlags)
	if err != nil {
		return fmt.Errorf("failed to parse var flags: %w", err)
	}

	// Load the template
	template, err := templates.LoadTemplate(cfg, templateFlag)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Get API key based on priority
	apiKey, err := getAPIKey(apiKeyFlag, cfg, template)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Add api_key to replacement variables if not empty
	if replaceVars == nil {
		replaceVars = make(map[string]string)
	}
	if apiKey != "" {
		replaceVars["api_key"] = apiKey
	}

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

	// Output result
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

// parseVarFlags parses --var flags with improved format support
func parseVarFlags(varFlags []string) (map[string]string, error) {
	replaceVars := make(map[string]string)

	for _, varFlag := range varFlags {
		// Support both name:value and name:type:value formats
		parts := strings.SplitN(varFlag, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid var format, expected name:value or name:type:value: %s", varFlag)
		}

		name := parts[0]
		if name == "" {
			return nil, fmt.Errorf("variable name cannot be empty in: %s", varFlag)
		}

		// Default to text type if only name:value provided
		var varType, value string
		if len(parts) == 2 {
			varType = "text"
			value = parts[1]
		} else {
			varType = parts[1]
			value = parts[2]
		}

		switch varType {
		case "text":
			replaceVars[name] = value

		case "file":
			if value == "" {
				return nil, fmt.Errorf("file path cannot be empty for variable %s", name)
			}
			fileContent, err := os.ReadFile(value)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s for variable %s: %w", value, name, err)
			}
			replaceVars[name] = string(fileContent)

		case "base64":
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

	// API key is optional - return empty string if no key is found
	return "", nil
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
