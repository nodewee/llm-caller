# LLM Caller

A CLI tool for calling various LLM (Large Language Model) services via HTTP requests using configurable JSON templates.

## Installation

### Download Binary

[Latest Release](https://github.com/nodewee/llm-caller/releases/latest)

### Build from Source
```bash
git clone https://github.com/nodewee/llm-caller.git
cd llm-caller
go build -o llm-caller
```

## Quick Start

```bash
# Set up your API key
export DEEPSEEK_API_KEY="sk-your-key-here"

# Download a template (supports both blob and raw URLs)
llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json

# Use the template
llm-caller call deepseek-chat --var "prompt:Explain quantum computing"

# For local LLMs (like Ollama) that don't require API keys
llm-caller call ollama-local --var "prompt:Explain quantum computing"
```

## Main Commands

LLM Caller provides the following command categories:

### 🔧 `call` - Execute LLM API Calls
Execute an LLM API call using a template. Supports three template sources:

1. **Template file**: `llm-caller call <template-name> --var name=value [options]`
2. **JSON string**: `llm-caller call --template-json '{"provider":"..."}' --var name=value [options]`
3. **Base64 encoded**: `llm-caller call --template-base64 "eyJ..." --var name=value [options]`

### 📝 `template` - Manage Templates  
Manage template files:
```bash
llm-caller template list                    # List available templates
llm-caller template download <github-url>   # Download from GitHub
llm-caller template show <template-name>    # Display template content
llm-caller template validate <template-name> # Validate template structure
```

### ⚙️ `config` - Configure Settings
Manage configuration:
```bash
llm-caller config <key>                     # Get configuration value
llm-caller config <key> <value>             # Set configuration
llm-caller config list                      # Show all settings
llm-caller config remove <key>              # Remove setting (revert to default)
```

### 🩺 `doctor` - Environment Check
Check configuration and environment:
```bash
llm-caller doctor                           # Diagnose setup issues
```

The doctor command checks:
- Configuration file existence and validity
- Template directories accessibility
- API keys availability (from file and environment variables)
- Template file integrity
- Provides specific recommendations to fix identified issues

### 🔍 `version` - Version Information
Display version and build information:
```bash
llm-caller version                          # Show detailed version info with commit hash and build time
llm-caller --version                        # Show detailed version info with commit hash and build time
```

## Configuration

Configuration is stored in `~/.llm-caller/config.yaml`. Available settings:

- `template_dir` - Directory where template files are stored
- `secret_file` - Path to JSON file containing API keys

## API Keys

API keys are checked in this order:
1. `--api-key` command line flag
2. Keys file (JSON format): `{"deepseek_api_key": "sk-xxx", "api_key": "sk-xxx"}`
3. Environment variables: `DEEPSEEK_API_KEY`, `API_KEY` (provider-specific keys are checked first)

API keys are optional for local LLMs like Ollama that don't require authentication.

Configure API keys file:
```bash
llm-caller config secret_file ~/.llm-caller/keys.json
```

## Templates

Templates are JSON files defining LLM API calls. Example:

```json
{
  "provider": "deepseek",
  "title": "DeepSeek Chat API",
  "description": "Template for calling DeepSeek's chat completion API",
  "request": {
    "url": "https://api.deepseek.com/chat/completions",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer {{api_key}}",
      "Content-Type": "application/json"
    },
    "body": {
      "model": "deepseek-chat",
      "messages": [{"role": "user", "content": "{{prompt}}"}]
    }
  },
  "response": {
    "path": "choices[0].message.content",
    "auto_detect": true
  }
}
```

## Template Sources

LLM Caller supports three mutually exclusive ways to provide templates:

### 1. Template Files (Traditional)
Templates are searched in:
1. Direct path (if contains `/` or `\`)
2. User template directory (configurable)
3. Downloaded templates (`~/.llm-caller/templates`)

```bash
llm-caller call deepseek-chat --var "prompt:Hello world"
```

### 2. JSON String Templates
Pass template content directly as a JSON string. Ideal for simple scenarios and quick testing:

```bash
llm-caller call --template-json '{
  "provider": "deepseek",
  "request": {
    "url": "https://api.deepseek.com/chat/completions",
    "headers": {"Authorization": "Bearer {{api_key}}"},
    "body": {
      "model": "deepseek-chat",
      "messages": [{"role": "user", "content": "{{prompt}}"}]
    }
  }
}' --var "prompt:Hello world"
```

### 3. Base64 Encoded Templates  
Perfect for automation, CI/CD, and complex templates with special characters:

```bash
# Generate Base64 template
TEMPLATE_JSON='{"provider":"deepseek",...}'
TEMPLATE_B64=$(echo "$TEMPLATE_JSON" | base64)

# Use in automation
llm-caller call --template-base64 "$TEMPLATE_B64" --var "prompt:$USER_INPUT"
```

**Benefits of Base64 encoding:**
- ✅ Solves shell escaping issues with quotes and special characters
- ✅ Perfect for CI/CD environments  
- ✅ Enables dynamic template generation in scripts
- ✅ Supports templates with newlines and complex formatting

### Template Structure

- `provider`: Service provider name (required)
- `title`: Human-readable title for the template (optional)
- `description`: Detailed description of the template (optional)
- `request`: HTTP request configuration (required)
  - `url`: API endpoint URL (required)
  - `method`: HTTP method (default: "POST")
  - `headers`: HTTP headers
  - `body`: Request body as JSON
- `response`: Response handling configuration
  - `path`: JSON path to extract text content (default: "choices[0].message.content")
  - `auto_detect`: Enable automatic response format detection (default: true)
  - `response_field_name`: Field name hint for auto-detection

## Usage Examples

### Basic Usage
```bash
# Using template file (traditional method)
llm-caller call deepseek-chat --var "prompt:Hello world"

# Using JSON template (direct inline)
llm-caller call --template-json '{"provider":"deepseek","request":{"url":"https://api.deepseek.com/chat/completions","headers":{"Authorization":"Bearer {{api_key}}"},"body":{"model":"deepseek-chat","messages":[{"role":"user","content":"{{prompt}}"}]}}}' --var "prompt:Hello world"

# Using Base64 template (for automation/scripting)
TEMPLATE_B64="eyJwcm92aWRlciI6ImRlZXBzZWVrIiwicmVxdWVzdCI6eyJ1cmwiOiJodHRwczovL2FwaS5kZWVwc2Vlay5jb20vY2hhdC9jb21wbGV0aW9ucyIsImhlYWRlcnMiOnsiQXV0aG9yaXphdGlvbiI6IkJlYXJlciB7e2FwaV9rZXl9fSJ9LCJib2R5Ijp7Im1vZGVsIjoiZGVlcHNlZWstY2hhdCIsIm1lc3NhZ2VzIjpbeyJyb2xlIjoidXNlciIsImNvbnRlbnQiOiJ7e3Byb21wdH19In1dfX19"
llm-caller call --template-base64 "$TEMPLATE_B64" --var "prompt:Hello world"

# Multiple variables (using colon-separated format)
llm-caller call translate --var "text:Hello" --var "target_lang:Chinese"
llm-caller call translate --var "text:text:Hello" --var "target_lang:text:Chinese"
```

### Variable Types
Variables support two types with the following formats:
- `name:value` - Simple format (shorthand for `name:text:value`)
- `name:type:value` - Detailed format with explicit type

Supported types:
- `text` - Use value as-is. If `value` is `-`, content is read raw from `stdin`.
- `file` - Reads content from a file path. The file content is used as a raw string. No special encoding (like Base64 for binary files) is performed.
  - If `path` is `-`, content is read raw from `stdin` without any conversion.

```bash
# Text (default and from stdin)
llm-caller call template --var "prompt:Hello world"
cat doc.txt | llm-caller call template --var "prompt:text:-"

# File content (reads as raw string)
# Reads my_document.txt as plain text
llm-caller call template --var "prompt:file:my_document.txt"

# Reads my_image.png as a raw string (no Base64 encoding)
llm-caller call vision-template --var "image_data:file:my_image.png"

# Pipe content from stdin (read as raw text)
cat my_image.png | llm-caller call vision-template --var "image_data:file:-"
```

### Output Options
```