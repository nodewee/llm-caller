package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-caller/pkg/templates"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// GenericClient is a generic HTTP client for calling LLM APIs
type GenericClient struct {
	APIKey string
	Client *http.Client
}

// NewGenericClient creates a new generic client
func NewGenericClient(apiKey string) (*GenericClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("empty API key provided")
	}

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
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response and extract the content
	result, err := c.extractResponseContent(body, template.Response.Path)
	if err != nil {
		return "", fmt.Errorf("failed to extract response content: %w", err)
	}

	return result, nil
}

// extractResponseContent extracts content from the response using the response path
func (c *GenericClient) extractResponseContent(body []byte, responsePath string) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// Navigate through the response path
	parts := strings.Split(responsePath, ".")
	current := interface{}(response)

	for _, part := range parts {
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
					return "", fmt.Errorf("field '%s' not found in response", arrayName)
				}
			}

			// Get the array element
			if arr, ok := current.([]interface{}); ok {
				if index >= len(arr) {
					return "", fmt.Errorf("array index %d out of bounds in response", index)
				}
				current = arr[index]
			} else {
				return "", fmt.Errorf("expected array but got %T for field '%s'", current, arrayName)
			}
		} else {
			// Regular field navigation
			current = navigateToField(current, part)
			if current == nil {
				return "", fmt.Errorf("field '%s' not found in response", part)
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
