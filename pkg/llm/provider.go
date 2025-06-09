package llm

import (
	"github.com/nodewee/llm-caller/pkg/templates"
)

// Provider is an interface for LLM providers
type Provider interface {
	Call(template *templates.Template) (string, error)
}

// GetProvider returns a generic provider for any template
func GetProvider(template *templates.Template, apiKey string) (Provider, error) {
	return NewGenericClient(apiKey)
}
