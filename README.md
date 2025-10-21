# Axon

Axon is a command-line tool that leverages the power LLMs to bring magic to your pipeline. It's designed to be a versatile and scriptable tool that can be easily integrated into your workflows.

## Examples

Here are some examples of how you can use Axon:

First you define a pattern in your config file (`~/.config/axon/axon.toml`):

```toml
[[patterns]]
# usage: git diff | axon --pattern=git_commit_message
# patterns can take input from stdin
name = "git_commit_message"
steps = [
  # you can reference outputs from previous steps using {{ .output_name }}
  # .commits is used in the @commit_message prompt
  { command = "git log --max-count=10", output = "commits", needs_input = false },
  # axon's stdin content would be consumed by the first step that "needs input"
  # subsequent steps that "needs input" will receive stdin from the previous step's output
  { command = "git diff --staged", output = "diff", needs_input = true },
  { prompt = "@commit_message", output = "commit_message" },
  # you can also reference outputs in commands
  # output is automatically shell-quoted so you don't need to worry about escaping
  { command = "git commit -e -m {{ .commit_message }}", needs_input = false },
]
```

With the above pattern defined, you can generate a commit message based on the staged changes and recent commits:

```sh
git add .
git diff --staged | axon --pattern=git_commit_message
```

For more examples, check out the [examples directory](./examples)

## Installation

```sh
go install github.com/madmaxieee/axon@latest
```

Setup shell completion:

```sh
axon completion fish > ~/.config/fish/completions/axon.fish
```

`axon` also comes with completions for `bash` and `zsh`.

## Configuration

Before you can start using Axon, you need to configure it with your OpenAI API key. Axon uses a configuration file located at `~/.config/axon/axon.toml`.

Here's the default configuration:

```toml
[general]
model = "openai/gpt-4o"

# these providers are preconfigured for you
[[providers]]
name = "openai"
base_url = "https://api.openai.com/v1"
api_key_env = "OPENAI_API_KEY"

[[providers]]
name = "google"
base_url = "https://generativelanguage.googleapis.com/v1beta"
api_key_env = "GOOGLE_API_KEY"

[[providers]]
name = "anthropic"
base_url = "https://api.anthropic.com/v1"
api_key_env = "ANTHROPIC_API_KEY"

[[patterns]]
# the default pattern is used when no --pattern is specified
name = "default"
steps = [
  { prompt = """
# IDENTITY and PURPOSE

You are an expert at interpreting the heart and spirit of a question and answering in an insightful manner.

# STEPS

- Deeply understand what's being asked.

- Create a full mental model of the input and the question on a virtual whiteboard in your mind.

# OUTPUT INSTRUCTIONS

- Do not output warnings or notes—just the requested sections.
""" },
]
```

## Acknowledgements

Axon was heavily inspired by these amazing projects:

- [mods](https://github.com/charmbracelet/mods)
  - The basic command line ergonomics and structure of Axon is inspired by mods.
- [fabirc](https://github.com/danielmiessler/Fabric)
  - The prompt format used in Axon is adapted from Fabric's prompt format.
  - You can actually point axon to fabrics pattern folders and use them directly!
