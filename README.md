# rmit (Open Router Auto Commit)
rmit is a CLI tool that uses the OpenRouter API to generate custom commit messages based on the changes in your git repository.

## Features
- Generate descriptive commit messages with AI
- Option to automatically commit changes
- Configuration management for API keys and settings

## Configuration

rmit supports storing configuration values such as API keys, API URL, and default model in a JSON configuration file in your home directory (`~/.rmitconfig`).

### Setting Configuration Values

Use the `set` command to configure rmit:

```bash
# Set your OpenRouter API key
rmit set api_key YOUR_API_KEY

# Set a custom API URL (optional)
rmit set api_url https://custom-endpoint.example.com/v1/chat/completions

# Set default model to use
rmit set default_model openai/gpt-4
```

### Getting Configuration Values

Use the `get` command to view your current configuration:

```bash
# View all configuration values
rmit get

# View specific configuration values
rmit get api_url
rmit get default_model
```

## Usage

### Basic Usage

Simply run `rmit` in your git repository to generate a commit message based on the current changes:

```bash
rmit
```

### Auto-Commit

Use the `-c` flag to automatically create a commit with the generated message:

```bash
rmit -c
```

### Custom Model

Specify a different model with the `-m` flag:

```bash
rmit -m openai/gpt-4
```