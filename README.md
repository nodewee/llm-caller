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
llm-caller -t deepseek-chat --var prompt:text:"Explain quantum computing"
```

## Configuration

Configuration is stored in `~/.llm-caller/config.yaml`. Manage it with:

```bash
llm-caller config list                           # Show all settings
llm-caller config set template_dir /my/templates # Set custom template directory
llm-caller config set secret_file /my/keys.json # Set API keys file
```

## API Keys

API keys are checked in this order:
1. `--api-key` command line flag
2. Keys file (JSON format): `{"deepseek_api_key": "sk-xxx", "api_key": "sk-xxx"}`
3. Environment variables: `DEEPSEEK_API_KEY`, `API_KEY`

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

### Template Management

```bash
# Download from GitHub
llm-caller template download https://github.com/owner/repo/blob/main/template.json

# List available templates
llm-caller template list
```

Templates are searched in:
1. Direct path (if contains `/` or `\`)
2. User template directory (configurable)
3. Downloaded templates (`~/.llm-caller/templates`)

## Usage

### Basic Usage
```bash
llm-caller -t <template> --var name:type:value [--api-key <key>] [-o <output-file>]
```

### Variable Types
- `text` - Use value as-is
- `file` - Read content from file
- `base64` - Decode base64 content

### Examples
```bash
# Simple text
llm-caller -t deepseek-chat --var prompt:text:"Hello world"

# From file
llm-caller -t translate --var text:file:document.txt -o translation.txt

# Multiple variables
llm-caller -t complex --var user:text:Alice --var doc:file:report.pdf

# Using different template paths
llm-caller -t ./my-template.json --var prompt:text:"Test"
llm-caller -t my-template --var prompt:text:"Test"  # Searches in configured directories
```

## Template Repositories

- [Official Templates](https://github.com/nodewee/llm-calling-templates) - Community-contributed templates

## License

[MIT License](LICENSE)