package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nodewee/llm-caller/pkg/config"
)

// RequestConfig contains the HTTP request configuration
type RequestConfig struct {
	URL     string                 `json:"url"`
	Method  string                 `json:"method,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body"`
}

// ResponseConfig contains the response parsing configuration
type ResponseConfig struct {
	Path string `json:"path,omitempty"`
}

// Template represents the unified template format
type Template struct {
	Provider string         `json:"provider"`
	Title    string         `json:"title,omitempty"`
	Request  RequestConfig  `json:"request"`
	Response ResponseConfig `json:"response,omitempty"`

	// Metadata fields for documentation (will be ignored during API calls)
	Description  string   `json:"description,omitempty"`
	APIDocument  string   `json:"api_document,omitempty"`
	Instructions []string `json:"instructions,omitempty"`
}

// Validate validates the template for required fields
func (t *Template) Validate() error {
	if t.Provider == "" {
		return fmt.Errorf("provider is required in template")
	}
	if t.Request.URL == "" {
		return fmt.Errorf("request.url is required in template")
	}
	if t.Request.Body == nil {
		return fmt.Errorf("request.body is required in template")
	}
	return nil
}

// LoadTemplate loads a template with priority order:
// 1. If templatePath is absolute or contains path separators, load directly
// 2. Otherwise, search in user configured template directory
// 3. Then search in default app config directory templates
func LoadTemplate(cfg *config.Config, templatePath string) (*Template, error) {
	// Automatically append .json extension if not present
	if !strings.HasSuffix(templatePath, ".json") {
		templatePath = templatePath + ".json"
	}

	// Check if it's a direct path (absolute or contains path separators)
	isDirectPath := filepath.IsAbs(templatePath) || strings.ContainsAny(templatePath, "/\\")

	if isDirectPath {
		// Normalize path for cross-platform compatibility
		templatePath = filepath.Clean(filepath.FromSlash(templatePath))
		data, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load template from direct path '%s': %w", templatePath, err)
		}
		return parseTemplate(data)
	}

	// For template names without path separators, search in directories
	var attemptedPaths []string

	// First, try user configured template directory
	userTemplateDir := cfg.GetString(config.KeyTemplateDir)
	if userTemplateDir != "" {
		userTemplatePath := filepath.Join(userTemplateDir, templatePath)
		attemptedPaths = append(attemptedPaths, userTemplatePath)
		if data, err := os.ReadFile(userTemplatePath); err == nil {
			return parseTemplate(data)
		}
	}

	// Second, try default app config templates directory
	defaultTemplateDir, err := config.GetDefaultTemplateDir()
	if err == nil {
		defaultTemplatePath := filepath.Join(defaultTemplateDir, templatePath)
		attemptedPaths = append(attemptedPaths, defaultTemplatePath)
		if data, err := os.ReadFile(defaultTemplatePath); err == nil {
			return parseTemplate(data)
		}
	}

	// If all attempts fail, return a descriptive error
	return nil, fmt.Errorf("template file not found, tried paths: %s", strings.Join(attemptedPaths, ", "))
}

// parseTemplate parses template data and applies defaults and validation
func parseTemplate(data []byte) (*Template, error) {
	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	// Set default values
	if template.Request.Method == "" {
		template.Request.Method = "POST"
	}
	if template.Response.Path == "" {
		template.Response.Path = "choices[0].message.content"
	}

	// Validate the template
	if err := template.Validate(); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	return &template, nil
}

// ReplaceVariables replaces variables in the template with values from the replacements map
func (t *Template) ReplaceVariables(replacements map[string]string) *Template {
	// Replace variables in request headers
	for key, value := range t.Request.Headers {
		t.Request.Headers[key] = replaceVariablesInString(value, replacements)
	}

	// Replace variables in request URL
	t.Request.URL = replaceVariablesInString(t.Request.URL, replacements)

	// Replace variables in request body
	t.Request.Body = replaceVariablesInInterface(t.Request.Body, replacements).(map[string]interface{})

	return t
}

// replaceVariablesInString replaces variables in a string
func replaceVariablesInString(content string, replacements map[string]string) string {
	result := content
	for key, value := range replacements {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", key), value)
	}
	return result
}

// replaceVariablesInInterface recursively replaces variables in any interface{} type
func replaceVariablesInInterface(data interface{}, replacements map[string]string) interface{} {
	switch v := data.(type) {
	case string:
		return replaceVariablesInString(v, replacements)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = replaceVariablesInInterface(value, replacements)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = replaceVariablesInInterface(item, replacements)
		}
		return result
	default:
		// For other types (numbers, booleans, etc.), return as-is
		return v
	}
}

// ListTemplates lists all JSON template files in the given directory
func ListTemplates(templateDir string) ([]string, error) {
	if templateDir == "" {
		return []string{}, nil
	}

	// Check if directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", templateDir, err)
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}
