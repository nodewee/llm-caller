package cmd

import (
	"fmt"
	"os"

	"github.com/nodewee/llm-caller/pkg/config"
	"github.com/nodewee/llm-caller/pkg/templates"
	"github.com/nodewee/llm-caller/pkg/utils"
	"github.com/spf13/cobra"
)

// Doctor command - diagnostic tool
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration and environment",
	Long: `Check the configuration and environment setup to ensure everything is working correctly.

This command verifies:
- Configuration file existence and validity
- Template directory accessibility
- API keys availability (from file and environment variables)
- Template file integrity

It will identify issues and provide specific recommendations for fixing them.`,
	RunE: runDoctor,
}

// runDoctor performs environment and configuration checks
func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 LLM Caller Environment Check")
	fmt.Println("================================")
	fmt.Println()

	var issues []string

	// Check config file
	configPath := cfg.GetConfigFilePath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		issues = append(issues, "Configuration file does not exist")
		fmt.Printf("❌ Config file: %s (not found)\n", configPath)
	} else {
		fmt.Printf("✅ Config file: %s\n", configPath)
	}

	// Check template directories
	userTemplateDir := cfg.GetString(config.KeyTemplateDir)
	if userTemplateDir != "" {
		if _, err := os.Stat(userTemplateDir); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("User template directory does not exist: %s", userTemplateDir))
			fmt.Printf("❌ User template dir: %s (not found)\n", userTemplateDir)
		} else {
			fmt.Printf("✅ User template dir: %s\n", userTemplateDir)
		}
	} else {
		fmt.Printf("ℹ️  User template dir: (not configured)\n")
	}

	defaultTemplateDir, err := config.GetDefaultTemplateDir()
	if err != nil {
		issues = append(issues, "Cannot determine default template directory")
		fmt.Printf("❌ Default template dir: (error)\n")
	} else {
		if _, err := os.Stat(defaultTemplateDir); os.IsNotExist(err) {
			fmt.Printf("⚠️  Default template dir: %s (will be created when needed)\n", defaultTemplateDir)
		} else {
			fmt.Printf("✅ Default template dir: %s\n", defaultTemplateDir)
		}
	}

	// Check API keys
	fmt.Println()
	fmt.Println("API Keys:")
	secretFile := cfg.GetString(config.KeySecretFile)
	if secretFile != "" {
		if _, err := os.Stat(secretFile); os.IsNotExist(err) {
			fmt.Printf("⚠️  Secret file: %s (not found)\n", secretFile)
		} else {
			if keys, err := loadApiKeys(secretFile); err == nil {
				fmt.Printf("✅ Secret file: %s (%d keys found)\n", secretFile, len(keys))
			} else {
				issues = append(issues, fmt.Sprintf("Secret file is invalid: %s", err))
				fmt.Printf("❌ Secret file: %s (invalid format)\n", secretFile)
			}
		}
	} else {
		fmt.Printf("ℹ️  Secret file: (not configured)\n")
	}

	// Check common environment variables
	envKeys := []string{"API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY", "ANTHROPIC_API_KEY"}
	foundEnvKeys := 0
	for _, key := range envKeys {
		if utils.GetEnvironmentVariableCaseInsensitive(key) != "" {
			foundEnvKeys++
		}
	}
	if foundEnvKeys > 0 {
		fmt.Printf("✅ Environment variables: %d API keys found\n", foundEnvKeys)
	} else {
		fmt.Printf("ℹ️  Environment variables: no API keys found (API keys are optional)\n")
	}

	// Check templates
	fmt.Println()
	fmt.Println("Templates:")
	var totalTemplates int
	if userTemplateDir != "" {
		if userTemplates, err := templates.ListTemplates(userTemplateDir); err == nil {
			totalTemplates += len(userTemplates)
			fmt.Printf("✅ User templates: %d found\n", len(userTemplates))
		}
	}
	if defaultTemplates, err := templates.ListTemplates(defaultTemplateDir); err == nil {
		totalTemplates += len(defaultTemplates)
		fmt.Printf("✅ Downloaded templates: %d found\n", len(defaultTemplates))
	}

	// Summary
	fmt.Println()
	fmt.Println("Summary:")
	if len(issues) == 0 {
		fmt.Printf("🎉 All checks passed! Found %d templates.\n", totalTemplates)
		fmt.Println()
		fmt.Println("Quick start:")
		fmt.Println("  llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json")
		fmt.Println("  llm-caller call deepseek-chat --var prompt=\"Hello world\" --api-key sk-xxx")
		fmt.Println("  # API key is optional:")
		fmt.Println("  llm-caller call ollama-local --var prompt=\"Hello world\"")
	} else {
		fmt.Printf("⚠️  Found %d issues:\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
		fmt.Println()
		fmt.Println("Recommendations:")
		fmt.Println("  - Run 'llm-caller config secret_file ~/.llm-caller/keys.json'")
		fmt.Println("  - Create API keys file with: {\"api_key\": \"sk-xxx\"}")
		fmt.Println("  - Download templates with: llm-caller template download <url>")
		fmt.Println("  - Remember: API keys are optional")
	}

	return nil
}
