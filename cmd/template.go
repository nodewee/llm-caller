package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nodewee/llm-caller/pkg/config"
	"github.com/nodewee/llm-caller/pkg/download"
	"github.com/nodewee/llm-caller/pkg/templates"
	"github.com/nodewee/llm-caller/pkg/utils"
	"github.com/spf13/cobra"
)

// Template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage template files",
	Long: `Manage template files including downloading, listing, viewing, and validating templates.

Templates define how to call LLM services and are stored in JSON format.
The system searches templates in user directory first, then downloaded templates.`,
}

// Template subcommands
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long:  `List all available template files from configured directories.`,
	RunE:  runTemplateList,
}

var templateDownloadCmd = &cobra.Command{
	Use:   "download <github-url>",
	Short: "Download a template from GitHub",
	Long: `Download a template file from a GitHub repository URL.

Supported URL formats:
  https://github.com/owner/repo/blob/branch/filename.json

Examples:
  llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateDownload,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Display template content",
	Long: `Display the content of a specified template file.

Examples:
  llm-caller template show deepseek-chat
  llm-caller template show deepseek-chat.json`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateShow,
}

var templateValidateCmd = &cobra.Command{
	Use:   "validate <template-name>",
	Short: "Validate template structure",
	Long: `Validate that a template file has correct JSON structure and required fields.

Examples:
  llm-caller template validate deepseek-chat
  llm-caller template validate my-template.json`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateValidate,
}

func init() {
	// Template subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateDownloadCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateValidateCmd)
}

// Template command handlers
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

	fmt.Printf("Template successfully downloaded to: %s\n", filePath)
	return nil
}

// checkTemplateExists checks if a template file exists before trying to load it
func checkTemplateExists(cfg *config.Config, templateName string) error {
	// Automatically append .json extension if not present
	if !strings.HasSuffix(templateName, ".json") {
		templateName = templateName + ".json"
	}

	// Check if it's a direct path (absolute or contains path separators)
	isDirectPath := filepath.IsAbs(templateName) || strings.ContainsAny(templateName, "/\\")

	if isDirectPath {
		// Normalize path for cross-platform compatibility
		templateName = filepath.Clean(filepath.FromSlash(templateName))
		if _, err := os.Stat(templateName); os.IsNotExist(err) {
			return fmt.Errorf("template file not found: %s", templateName)
		}
		return nil
	}

	// For template names without path separators, search in directories
	var exists bool

	// First, try user configured template directory
	userTemplateDir := cfg.GetString(config.KeyTemplateDir)
	if userTemplateDir != "" {
		userTemplatePath := filepath.Join(userTemplateDir, templateName)
		if _, err := os.Stat(userTemplatePath); err == nil {
			exists = true
		}
	}

	// Second, try default app config templates directory
	if !exists {
		defaultTemplateDir, err := config.GetDefaultTemplateDir()
		if err == nil {
			defaultTemplatePath := filepath.Join(defaultTemplateDir, templateName)
			if _, err := os.Stat(defaultTemplatePath); err == nil {
				exists = true
			}
		}
	}

	if !exists {
		return fmt.Errorf("template file not found: %s", templateName)
	}

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// First check if the template exists
	if err := checkTemplateExists(cfg, templateName); err != nil {
		return err
	}

	// Load the template
	template, err := templates.LoadTemplate(cfg, templateName)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Pretty print the template as JSON
	jsonData, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func runTemplateValidate(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// First check if the template exists
	if err := checkTemplateExists(cfg, templateName); err != nil {
		return err
	}

	// Try to load and validate the template
	template, err := templates.LoadTemplate(cfg, templateName)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Additional validation
	if err := template.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	fmt.Printf("✅ Template '%s' is valid\n", templateName)
	fmt.Printf("Provider: %s\n", template.Provider)
	fmt.Printf("URL: %s\n", template.Request.URL)
	fmt.Printf("Method: %s\n", template.Request.Method)

	if template.Title != "" {
		fmt.Printf("Title: %s\n", template.Title)
	}
	if template.Description != "" {
		fmt.Printf("Description: %s\n", template.Description)
	}

	return nil
}
