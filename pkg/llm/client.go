package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nodewee/llm-caller/pkg/templates"
)

// GenericClient is a generic HTTP client for calling LLM APIs
type GenericClient struct {
	APIKey string
	Client *http.Client
}

// NewGenericClient creates a new generic client
func NewGenericClient(apiKey string) (*GenericClient, error) {
	// Allow empty API key for local LLMs that don't require authentication
	return &GenericClient{
		APIKey: apiKey,
		Client: &http.Client{},
	}, nil
}

// Call calls the LLM API with the given template
func (c *GenericClient) Call(template *templates.Template) (string, error) {
	// Marshal the request body to JSON
	reqBytes, err := json.Marshal(template.Request.Body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(template.Request.Method, template.Request.URL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from template
	for key, value := range template.Request.Headers {
		httpReq.Header.Set(key, value)
	}

	// Always add/overwrite User-Agent header
	httpReq.Header.Set("User-Agent", "https://github.com/nodewee/llm-caller")

	// Send the request
	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Use auto-detection if enabled, otherwise use the specified path
	var result string
	if template.Response.AutoDetect {
		result, err = c.autoDetectResponseContent(body, template.Response.ResponseFieldName)
		if err != nil {
			// Fall back to path-based extraction if auto-detection fails
			result, err = c.extractResponseContentByPath(body, template.Response.Path)
			if err != nil {
				// Preserve the detailed error from extractResponseContentByPath
				return "", err
			}
		}
	} else {
		// Use path-based extraction directly
		result, err = c.extractResponseContentByPath(body, template.Response.Path)
		if err != nil {
			// Preserve the detailed error from extractResponseContentByPath
			return "", err
		}
	}

	return result, nil
}

// autoDetectResponseContent tries to automatically detect the response format
func (c *GenericClient) autoDetectResponseContent(body []byte, preferredResponseField string) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// If a specific response field is requested, try that first
	if preferredResponseField != "" {
		if content, ok := response[preferredResponseField]; ok {
			if strContent, ok := content.(string); ok {
				return strContent, nil
			}
		}
	}

	// Otherwise use the general detection logic
	content, ok := detectResponseFormat(response)
	if !ok {
		return "", fmt.Errorf("couldn't auto-detect response format")
	}

	return content, nil
}

// extractResponseContentByPath extracts content from the response using a dot-notation path
// This is the original path-based extraction logic
func (c *GenericClient) extractResponseContentByPath(body []byte, responsePath string) (string, error) {
	if responsePath == "" {
		return "", fmt.Errorf("response path is required for extraction")
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// Navigate through the response path
	parts := strings.Split(responsePath, ".")
	current := interface{}(response)
	var pathSoFar string

	for i, part := range parts {
		pathSoFar = strings.Join(parts[:i+1], ".")

		// Handle array indices like "choices[0]"
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			arrayName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return "", fmt.Errorf("invalid array index '%s' in response path", indexStr)
			}

			// Navigate to the array
			if arrayName != "" {
				current = navigateToField(current, arrayName)
				if current == nil {
					// Show error with response structure for better debugging
					prettyResponse, _ := formatResponseStructure(response)
					return "", fmt.Errorf("field '%s' not found in response path '%s'. API response structure: %s",
						arrayName, pathSoFar, prettyResponse)
				}
			}

			// Get the array element
			if arr, ok := current.([]interface{}); ok {
				if index >= len(arr) {
					prettyResponse, _ := formatResponseStructure(response)
					return "", fmt.Errorf("array index %d out of bounds in response path '%s' (array length: %d). API response structure: %s",
						index, pathSoFar, len(arr), prettyResponse)
				}
				current = arr[index]
			} else {
				prettyResponse, _ := formatResponseStructure(response)
				return "", fmt.Errorf("expected array but got %T for field '%s' in path '%s'. API response structure: %s",
					current, arrayName, pathSoFar, prettyResponse)
			}
		} else {
			// Regular field navigation
			current = navigateToField(current, part)
			if current == nil {
				// Show error with response structure for better debugging
				prettyResponse, _ := formatResponseStructure(response)
				return "", fmt.Errorf("field '%s' not found in response path '%s'. API response structure: %s",
					part, pathSoFar, prettyResponse)
			}
		}
	}

	// Convert the final result to string
	if str, ok := current.(string); ok {
		return str, nil
	}

	// If it's not a string, try to convert it
	return fmt.Sprintf("%v", current), nil
}

// formatResponseStructure returns a formatted string representation of the response structure
// It's used for debugging when a path can't be found
func formatResponseStructure(response map[string]interface{}) (string, error) {
	// Pretty-print the response structure with indent
	prettyJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "{error formatting response}", err
	}

	// If the response is too large, truncate it
	if len(prettyJSON) > 1000 {
		return string(prettyJSON[:1000]) + "... (truncated)", nil
	}

	return string(prettyJSON), nil
}

// detectResponseFormat attempts to identify and extract content from common LLM response formats
// Handles various formats from different providers:
// - Ollama (legacy and newer versions)
// - OpenAI API (chat completions and standard completions)
// - Anthropic
// - Cohere
// - Claude
// Returns the extracted content string and a boolean success indicator
func detectResponseFormat(response map[string]interface{}) (string, bool) {
	// Ollama format (new) - direct "response" field
	// {"model":"qwen2.5vl","created_at":"...","response":"Hello!...","done":true}
	if response, ok := response["response"]; ok {
		if strResponse, ok := response.(string); ok {
			return strResponse, true
		}
	}

	// OpenAI format - choices[0].message.content or choices[0].text
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		choice := choices[0]
		if choiceMap, ok := choice.(map[string]interface{}); ok {
			// ChatCompletions format (choices[0].message.content)
			if message, ok := choiceMap["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, true
				}
			}
			// Completions format (choices[0].text)
			if text, ok := choiceMap["text"].(string); ok {
				return text, true
			}
		}
	}

	// Anthropic format - content or completion
	if content, ok := response["content"].(string); ok {
		return content, true
	}
	if completion, ok := response["completion"].(string); ok {
		return completion, true
	}

	// Cohere format - generations[0].text
	if generations, ok := response["generations"].([]interface{}); ok && len(generations) > 0 {
		generation := generations[0]
		if genMap, ok := generation.(map[string]interface{}); ok {
			if text, ok := genMap["text"].(string); ok {
				return text, true
			}
		}
	}

	// Ollama format (older version) - content
	if content, ok := response["content"].(string); ok {
		return content, true
	}

	// Claude format - content[0].text
	if content, ok := response["content"].([]interface{}); ok && len(content) > 0 {
		if contentMap, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentMap["text"].(string); ok {
				return text, true
			}
		}
	}

	// No recognized format
	return "", false
}

// navigateToField navigates to a specific field in a map or struct
func navigateToField(data interface{}, field string) interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return v[field]
	case map[interface{}]interface{}:
		return v[field]
	default:
		// Use reflection for struct fields
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			fieldVal := val.FieldByName(field)
			if fieldVal.IsValid() {
				return fieldVal.Interface()
			}
		}
		return nil
	}
}
