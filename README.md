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
llm-caller call deepseek-chat --var prompt="Explain quantum computing"

# For local LLMs (like Ollama) that don't require API keys
llm-caller call ollama-local --var prompt="Explain quantum computing"
```

## Main Commands

LLM Caller provides the following command categories:

### 🔧 `call` - Execute LLM API Calls
Execute an LLM API call using a template:
```bash
llm-caller call <template> --var name=value [options]
```

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
llm-caller --version                        # Show basic version number
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

Templates are searched in:
1. Direct path (if contains `/` or `\`)
2. User template directory (configurable)
3. Downloaded templates (`~/.llm-caller/templates`)

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
# Simple text variable
llm-caller call deepseek-chat --var "prompt:Hello world"

# Multiple variables (using colon-separated format)
llm-caller call translate --var "text:Hello" --var "target_lang:Chinese"
llm-caller call translate --var "text:text:Hello" --var "target_lang:text:Chinese"
```

### Variable Types
Variables support three types with the following formats:
- `name:value` - Simple format (text type by default)
- `name:type:value` - Detailed format with explicit type

Supported types:
- `text` - Use value as-is (default if type not specified)
- `file` - Read content from file
- `base64` - Decode base64 content

```bash
# Text (default)
llm-caller call template --var "prompt:Hello world"
llm-caller call template --var "prompt:text:Hello world"

# File content
llm-caller call translate --var "prompt:file:document.txt"

# Base64 content
llm-caller call analyze --var "prompt:base64:SGVsbG8gd29ybGQ="
```

### Output Options
```bash
# Print to stdout (default)
llm-caller call deepseek-chat --var prompt="Hello"

# Save to file
llm-caller call translate --var text:file:doc.txt -o translation.txt
```

### Template Management
```bash
# Download templates from GitHub (blob URLs)
llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json

# Download templates from GitHub (raw URLs)
llm-caller template download https://raw.githubusercontent.com/nodewee/llm-calling-templates/refs/heads/main/ollama-image-class.json

# List all available templates
llm-caller template list

# View template content
llm-caller template show deepseek-chat

# Validate template structure
llm-caller template validate my-template
```

### Configuration Examples
```bash
# Set custom template directory
llm-caller config template_dir ~/my-templates

# Set API keys file
llm-caller config secret_file ~/.api-keys.json

# View current configuration
llm-caller config list

# Get specific setting
llm-caller config template_dir

# Reset to default
llm-caller config remove template_dir
```

### Troubleshooting
```bash
# Check environment and configuration
llm-caller doctor

# This will verify:
# - Configuration file existence
# - Template directory accessibility  
# - API keys availability
# - Template file integrity
```

## Template Repositories

- [Official Templates](https://github.com/nodewee/llm-calling-templates) - Community-contributed templates

## License

[MIT License](LICENSE)

### Template Validation

To validate a template structure:
```bash
llm-caller template validate my-template
```

This checks for:
- Valid JSON format
- Required fields (provider, request URL, request body)
- Proper structure for HTTP requests
- Response handling configuration