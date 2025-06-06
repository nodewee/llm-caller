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

# Download a template
llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json

# Use the template
llm-caller call deepseek-chat --var prompt="Explain quantum computing"
```

## Main Commands

LLM Caller provides four main command categories:

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
llm-caller template show <name>             # Display template content
llm-caller template validate <name>         # Validate template structure
```

### ⚙️ `config` - Configure Settings
Manage configuration:
```bash
llm-caller config set <key> <value>         # Set configuration
llm-caller config get <key>                 # Get configuration value
llm-caller config list                      # Show all settings
llm-caller config unset <key>               # Remove setting (revert to default)
```

### 🩺 `doctor` - Environment Check
Check configuration and environment:
```bash
llm-caller doctor                           # Diagnose setup issues
```

## Configuration

Configuration is stored in `~/.llm-caller/config.yaml`. Available settings:

- `template_dir` - Directory where template files are stored
- `secret_file` - Path to JSON file containing API keys

## API Keys

API keys are checked in this order:
1. `--api-key` command line flag
2. Keys file (JSON format): `{"deepseek_api_key": "sk-xxx", "api_key": "sk-xxx"}`
3. Environment variables: `DEEPSEEK_API_KEY`, `API_KEY`

Configure API keys file:
```bash
llm-caller config set secret_file ~/.llm-caller/keys.json
```

## Templates

Templates are JSON files defining LLM API calls. Example:

```json
{
  "provider": "deepseek",
  "request": {
    "url": "https://api.deepseek.com/chat/completions",
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
    "path": "choices[0].message.content"
  }
}
```

Templates are searched in:
1. Direct path (if contains `/` or `\`)
2. User template directory (configurable)
3. Downloaded templates (`~/.llm-caller/templates`)

## Usage Examples

### Basic Usage
```bash
# Simple text variable
llm-caller call deepseek-chat --var prompt="Hello world"

# Multiple variables (supports both formats)
llm-caller call translate --var text="Hello" --var target_lang="Chinese"
llm-caller call translate --var text:text:"Hello" --var target_lang:text:"Chinese"
```

### Variable Types
Variables support three types:
- `text` - Use value as-is (default if type not specified)
- `file` - Read content from file
- `base64` - Decode base64 content

```bash
# Text (default)
llm-caller call template --var prompt="Hello world"
llm-caller call template --var prompt:text:"Hello world"

# File content
llm-caller call translate --var text:file:document.txt

# Base64 content
llm-caller call analyze --var data:base64:SGVsbG8gd29ybGQ=
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
# Download templates from GitHub
llm-caller template download https://github.com/nodewee/llm-calling-templates/blob/main/deepseek-chat.json

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
llm-caller config set template_dir ~/my-templates

# Set API keys file
llm-caller config set secret_file ~/.api-keys.json

# View current configuration
llm-caller config list

# Get specific setting
llm-caller config get template_dir

# Reset to default
llm-caller config unset template_dir
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