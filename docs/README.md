# templr Documentation

Welcome to the templr documentation! This guide will help you master templr, a powerful Go-based templating CLI inspired by Helm.

## 📚 Documentation Index

### Getting Started
- **[Installation Guide](installation.md)** - Download, install, and verify templr
- **[Quick Start](#quick-start)** - Get up and running in 5 minutes
- **[CLI Reference](cli-reference.md)** - Complete command and flag documentation

### Core Concepts
- **[Templating Guide](templating-guide.md)** - Complete template syntax and features
- **[Configuration Files](configuration.md)** - `.templr.yaml` configuration reference
- **[Examples](examples.md)** - Real-world use cases and patterns

### Reference
- **[CLI Reference](cli-reference.md)** - All commands, flags, and exit codes
- **[Configuration Reference](configuration.md#configuration-options-reference)** - Complete config option tables
- **[Main README](../README.md)** - Project overview and features

## Quick Start

### 1. Install templr

```bash
# macOS/Linux via Homebrew
brew tap kanopi/templr
brew install templr

# Or via curl
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash

# Or via Docker
docker pull kanopi/templr
```

### 2. Create Your First Template

**template.tpl:**
```yaml
name: {{ .name }}
environment: {{ .environment }}
version: {{ .version }}
```

**values.yaml:**
```yaml
name: myapp
environment: production
version: 1.0.0
```

### 3. Render the Template

```bash
# Render to stdout
templr render -in template.tpl -data values.yaml

# Render to file
templr render -in template.tpl -data values.yaml -out config.yaml
```

### 4. Validate Templates

```bash
# Check for errors before rendering
templr lint -i template.tpl -d values.yaml
```

## Learning Path

### For New Users
1. Start with [Installation Guide](installation.md)
2. Follow the Quick Start above
3. Read [Templating Guide](templating-guide.md) sections 1-5
4. Try [Basic Examples](examples.md#single-file-rendering)

### For Teams
1. Read [Configuration Files](configuration.md)
2. Review [Team Workflow Examples](examples.md#team-workflows)
3. Set up [Project Configuration](configuration.md#project-structure-best-practices)
4. Implement [CI/CD Integration](examples.md#cicd-integration)

### For Advanced Users
1. Explore [Advanced Templating](templating-guide.md#advanced-capabilities-and-sprig-functions)
2. Study [Multi-Environment Setups](configuration.md#multi-environment-setup)
3. Implement [Custom Workflows](examples.md#advanced-patterns)
4. Review [Security Best Practices](configuration.md#security-focused-project)

## Common Tasks

### Rendering Templates

```bash
# Single file
templr render -in template.tpl -data values.yaml -out output.txt

# Directory of templates
templr dir --dir templates/ -in main.tpl -data values.yaml -out output.txt

# Walk entire directory tree
templr walk --src templates/ --dst output/
```

See: [CLI Reference](cli-reference.md) | [Examples](examples.md)

### Linting and Validation

```bash
# Lint single file
templr lint -i template.tpl -d values.yaml

# Lint entire directory
templr lint --src templates/ -d values.yaml

# Fail on warnings in CI
templr lint --src templates/ -d values.yaml --fail-on-warn
```

See: [Linting Guide](cli-reference.md#lint-command) | [CI/CD Examples](examples.md#cicd-integration)

### Configuration Management

```bash
# Use project config (.templr.yaml in current directory)
templr lint --src templates/

# Use specific config
templr lint --config .templr.prod.yaml --src templates/

# Use user config (~/.config/templr/config.yaml)
# Automatically loaded
```

See: [Configuration Guide](configuration.md) | [Configuration Examples](configuration.md#configuration-use-cases)

## Key Features

### Powerful Templating
- **Helm-like syntax** - Familiar to Kubernetes users
- **Sprig functions** - 100+ built-in template functions
- **`.Files` API** - Access files during template rendering
- **Custom delimiters** - Avoid conflicts with other systems

See: [Templating Guide](templating-guide.md)

### Flexible Modes
- **Single-file** - Render individual templates
- **Directory** - Render with shared helpers
- **Walk** - Process entire directory trees
- **Lint** - Validate without rendering

See: [CLI Reference](cli-reference.md)

### Configuration Files
- **Project config** - `.templr.yaml` in your project
- **User config** - `~/.config/templr/config.yaml` for personal preferences
- **Hierarchical** - CLI flags > Project > User > Defaults

See: [Configuration Guide](configuration.md)

### Developer Experience
- **Guard strings** - Prevent accidental overwrites
- **Dry-run mode** - Preview changes safely
- **Exit codes** - CI/CD friendly error handling
- **Multiple formats** - JSON, GitHub Actions output

See: [CLI Reference](cli-reference.md#exit-codes)

## Contributing

We welcome contributions! See the [main README](../README.md) for contribution guidelines.

## Support

- **Issues**: [GitHub Issues](https://github.com/kanopi/templr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kanopi/templr/discussions)
- **License**: [MIT](../LICENSE)

## Version

This documentation is for templr version 1.x. For older versions, see the [releases page](https://github.com/kanopi/templr/releases).
