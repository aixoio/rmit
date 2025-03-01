# rmit (Open Router Auto Commit)

rmit is a CLI tool that uses the OpenRouter API to generate custom commit messages based on the changes in your git repository. It analyzes your git diff and uses AI to create meaningful, conventional commit messages.

## Features

- Generate descriptive commit messages with AI
- Option to automatically commit changes
- Configuration management for API keys and settings
- Interactive mode with options to refine commit messages
- Support for conventional commit format
- Project type detection for context-aware commit messages

## Installation

### Prerequisites

- Go 1.18 or higher
- Git

### From Source

```bash
# Clone the repository
git clone https://github.com/aixoio/rmit.git
cd rmit

# Build the binary
go build -o rmit .

# Move to a directory in your PATH (optional)
sudo mv rmit /usr/local/bin/
```

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

### Environment Variables

You can also set your API key using an environment variable:

```bash
export OPENROUTER_API_KEY=your_api_key_here
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

### Interactive Options

When running without the auto-commit flag, rmit provides an interactive interface with the following options:

- `y/yes` - Create commit with the generated message
- `n/no` - Cancel commit
- `g` - Generate a more detailed commit message
- `r` - Retry with a new generation
- `s` - Summarize the message (make it shorter)
- `p` - Provide feedback for the message (custom prompt)

Example workflow:

```bash
# Make some changes to your code
# Run rmit to generate a commit message
rmit

# If you want a more detailed message
g

# If you want to provide specific feedback
p
> Focus on the performance improvements

# Accept the message
y
```

## How It Works

1. rmit detects changes in your git repository (staged or unstaged)
2. It analyzes the diff and identifies changed files
3. It detects the project type (Go, JavaScript, Java, etc.) for better context
4. It sends this information to the OpenRouter API with a prompt for a conventional commit message
5. It presents the generated message with options to accept, refine, or reject it

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.