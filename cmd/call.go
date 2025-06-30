package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	varFlags           []string
	apiKeyFlag         string
	outputFlag         string
	templateJSONFlag   string
	templateBase64Flag string
)

// Call command - main functionality
var callCmd = &cobra.Command{
	Use:   "call [<template>]",
	Short: "Execute an LLM API call using a template",
	Long: `Execute an LLM API call using a specified template with variable substitution.

This command loads a template, replaces variables, and makes an HTTP request to the LLM service.
The result is either printed to stdout or saved to a file.

Template Sources (mutually exclusive):
1. Template file: llm-caller call <template-name>
2. JSON string: llm-caller call --template-json '{"provider":"..."}'
3. Base64 encoded: llm-caller call --template-base64 "eyJ..."

Variable Types & Data Handling:
- name:value (default type is 'text')
- name:type:value, where type is one of:
  - text: Use value as-is. If value is '-', read raw content from stdin.
  - file: Reads content from a file path. The file content is used as a raw string without any special encoding.
    - If path is '-', reads raw content from stdin.

API keys are checked in this order:
1. --api-key command line flag
2. Keys file (configured with 'config secret_file')
3. Environment variables (provider-specific keys checked first)

API keys are optional for local LLMs like Ollama that don't require authentication.

Examples:
  # Using template file
  llm-caller call deepseek-chat --var "prompt:Hello world"
  
  # Handle large data via file
  llm-caller call open-chat --var "image:file:./image.png"

  # Pipe content from stdin
  cat README.md | llm-caller call my-template --var "prompt:text:-"
  cat image.png | llm-caller call my-template --var "image:file:-"
  
  # Using JSON string
  llm-caller call --template-json '{"provider":"deepseek","request":{"url":"https://api.deepseek.com/chat/completions","headers":{"Authorization":"Bearer {{api_key}}"},"body":{"model":"deepseek-chat","messages":[{"role":"user","content":"{{prompt}}"}]}}}' --var "prompt:Hello world"
  
  # Using Base64 encoded template (for complex templates or scripting)
  llm-caller call --template-base64 "eyJwcm92aWRlciI6ImRlZXBzZWVrIiwicmVxdWVzdCI6eyJ1cmwiOiJodHRwczovL2FwaS5kZWVwc2Vlay5jb20vY2hhdC9jb21wbGV0aW9ucyIsImhlYWRlcnMiOnsiQXV0aG9yaXphdGlvbiI6IkJlYXJlciB7e2FwaV9rZXl9fSJ9LCJib2R5Ijp7Im1vZGVsIjoiZGVlcHNlZWstY2hhdCIsIm1lc3NhZ2VzIjpbeyJyb2xlIjoidXNlciIsImNvbnRlbnQiOiJ7e3Byb21wdH19In1dfX19" --var "prompt:Hello world"
  
  # Local LLM (API key optional)
  llm-caller call ollama-local --var "prompt:Tell me a joke"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCall,
}

func init() {
	// Call command flags
	callCmd.Flags().StringArrayVar(&varFlags, "var", []string{}, "Variable in 'name[:type]:value' format (e.g., 'prompt:file:my.txt'). Type can be 'text' or 'file'. Use '-' to read from stdin.")
	callCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key (optional, overrides config and environment)")
	callCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path (default: stdout)")
	callCmd.Flags().StringVar(&templateJSONFlag, "template-json", "", "Template as JSON string (mutually exclusive with template file and --template-base64)")
	callCmd.Flags().StringVar(&templateBase64Flag, "template-base64", "", "Template as Base64 encoded JSON (mutually exclusive with template file and --template-json)")
}

// runCall handles the call command
func runCall(cmd *cobra.Command, args []string) error {
	// Validate template source arguments (mutually exclusive)
	templateSources := 0
	var templateFlag string

	if len(args) > 0 && args[0] != "" {
		templateSources++
		templateFlag = args[0]
	}
	if cmd.Flags().Changed("template-json") {
		templateSources++
	}
	if cmd.Flags().Changed("template-base64") {
		templateSources++
	}

	if templateSources == 0 {
		return fmt.Errorf("must specify a template source: template file, --template-json, or --template-base64")
	}
	if templateSources > 1 {
		return fmt.Errorf("template sources are mutually exclusive: specify only one of template file, --template-json, or --template-base64")
	}

	// Parse var flags with improved format support
	replaceVars, err := parseVarFlags(varFlags)
	if err != nil {
		return fmt.Errorf("failed to parse var flags: %w", err)
	}

	// Load the template based on the source type
	var template *templates.Template
	if templateFlag != "" {
		// Load from file (existing logic)
		template, err = templates.LoadTemplate(cfg, templateFlag)
		if err != nil {
			return fmt.Errorf("failed to load template: %w", err)
		}
	} else if cmd.Flags().Changed("template-json") {
		// Load from JSON string
		if templateJSONFlag == "" {
			return fmt.Errorf("--template-json cannot be empty")
		}
		template, err = templates.LoadTemplateFromJSON(templateJSONFlag)
		if err != nil {
			return fmt.Errorf("failed to parse template JSON: %w", err)
		}
	} else if cmd.Flags().Changed("template-base64") {
		// Load from Base64 encoded JSON
		if templateBase64Flag == "" {
			return fmt.Errorf("--template-base64 cannot be empty")
		}
		jsonData, err := base64.StdEncoding.DecodeString(templateBase64Flag)
		if err != nil {
			return fmt.Errorf("failed to decode Base64 template: %w", err)
		}
		template, err = templates.LoadTemplateFromJSON(string(jsonData))
		if err != nil {
			return fmt.Errorf("failed to parse Base64 decoded template JSON: %w", err)
		}
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
			if value == "-" {
				stdinContent, err := io.ReadAll(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("failed to read from stdin for variable %s: %w", name, err)
				}
				replaceVars[name] = string(stdinContent)
			} else {
				replaceVars[name] = value
			}
		case "file":
			var content []byte
			var err error
			if value == "-" {
				// Read raw content from stdin
				content, err = io.ReadAll(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("failed to read from stdin for variable %s: %w", name, err)
				}
			} else {
				// Read raw content from file path
				if value == "" {
					return nil, fmt.Errorf("file path cannot be empty for variable %s", name)
				}
				content, err = os.ReadFile(value)
				if err != nil {
					return nil, fmt.Errorf("failed to read file %s for variable %s: %w", value, name, err)
				}
			}
			replaceVars[name] = string(content)

		default:
			return nil, fmt.Errorf("unsupported variable type '%s' for variable %s, supported types: text, file", varType, name)
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
