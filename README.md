# templr

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/kanopi/templr/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/kanopi/templr/tree/main) [![Docker Pulls](https://img.shields.io/docker/pulls/kanopi/templr)](https://hub.docker.com/r/kanopi/templr) [![Latest Release](https://img.shields.io/github/v/release/kanopi/templr)](https://github.com/kanopi/templr/releases) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

## Overview

templr is a powerful Go-based templating CLI inspired by Helm and Go's `text/template` package. Render templates from single files or entire directories with advanced features like linting, validation, and configuration management.

**Perfect for:**
- Generating Kubernetes manifests
- Creating configuration files
- Building documentation
- CI/CD pipelines
- Infrastructure as Code

## Key Features

✨ **Flexible Rendering** - Single files, directories, or recursive tree walking
🔍 **Lint Mode** - Validate templates without rendering
🛡️ **Guard Protection** - Prevent accidental overwrites
📝 **Configuration Files** - Project and user-level `.templr.yaml` support
🎯 **Strict Validation** - Catch undefined variables and errors early
🔧 **100+ Functions** - Full Sprig function library included
📁 **`.Files` API** - Access files during template rendering
🚀 **CI/CD Ready** - Exit codes, JSON output, GitHub Actions support

## Quick Start

### Installation

```bash
# macOS/Linux via Homebrew
brew tap kanopi/templr
brew install templr

# Or via curl
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash

# Or via Docker
docker pull kanopi/templr
```

**[Complete Installation Guide →](docs/installation.md)**

### Basic Usage

```bash
# Render a single template
templr render -in template.tpl -data values.yaml -out output.txt

# Render entire directory tree
templr walk --src templates/ --dst output/

# Validate templates
templr lint --src templates/ -d values.yaml
```

### Example

**template.tpl:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .app.name }}
  namespace: {{ .namespace }}
data:
  environment: {{ .environment }}
  version: {{ .app.version }}
```

**values.yaml:**
```yaml
app:
  name: myapp
  version: "1.0.0"
namespace: production
environment: prod
```

**Render:**
```bash
templr render -in template.tpl -data values.yaml -out config.yaml
```

## Documentation

### 📖 Guides

- **[Documentation Hub](docs/README.md)** - Complete documentation index
- **[Templating Guide](docs/templating-guide.md)** - Template syntax and features
- **[Configuration Files](docs/configuration.md)** - `.templr.yaml` reference
- **[CLI Reference](docs/cli-reference.md)** - All commands and flags
- **[Examples](docs/examples.md)** - Real-world use cases

### 🚀 Getting Started

- **[Installation Guide](docs/installation.md)** - Platform-specific installation
- **[Quick Start](docs/README.md#quick-start)** - Get up and running
- **[Basic Examples](docs/examples.md)** - Simple use cases

### 📚 Advanced Topics

- **[Team Workflows](docs/configuration.md#team-workflow-with-user-and-project-configs)** - Multi-developer setups
- **[CI/CD Integration](docs/examples.md#cicd-integration)** - GitHub Actions, GitLab, CircleCI
- **[Multi-Environment](docs/configuration.md#multi-environment-setup)** - Dev/staging/prod configs

## Common Commands

### Rendering

```bash
# Single file
templr render -in template.tpl -data values.yaml -out output.txt

# Directory with helpers
templr dir --dir templates/ -in main.tpl -data values.yaml -out output.txt

# Walk entire tree
templr walk --src templates/ --dst output/ -data values.yaml
```

### Validation

```bash
# Lint templates
templr lint --src templates/ -d values.yaml

# Strict validation (fail on warnings)
templr lint --src templates/ -d values.yaml --fail-on-warn

# JSON output for CI/CD
templr lint --src templates/ -d values.yaml --format json
```

### Configuration

```bash
# Use project config (.templr.yaml)
templr lint --src templates/

# Use specific config
templr lint --config .templr.prod.yaml --src templates/

# User config is automatically loaded from ~/.config/templr/config.yaml
```

**[Complete CLI Reference →](docs/cli-reference.md)**

## Configuration Files

Create a `.templr.yaml` in your project root to set defaults and enforce policies:

```yaml
# File handling
files:
  extensions: [tpl, yaml, md]
  default_values_file: ./values.yaml

# Linting rules
lint:
  fail_on_warn: true
  fail_on_undefined: true
  required_vars:
    - name
    - version
  disallow_functions:
    - env                # Block environment access
    - exec               # Block command execution

# Rendering defaults
render:
  inject_guard: true
  guard_string: "#templr generated"
```

**Benefits:**
- ✅ Shorter commands
- ✅ Version-controlled settings
- ✅ Team consistency
- ✅ Security policies

**[Complete Configuration Guide →](docs/configuration.md)**

## Use Cases

### Kubernetes Manifests

```bash
# Generate and validate manifests
templr lint --src k8s/templates/ -d values.prod.yaml --fail-on-warn
templr walk --src k8s/templates/ --dst manifests/ -data values.prod.yaml
kubectl apply -f manifests/
```

### Documentation Generation

```bash
# Generate docs from templates
templr walk --src docs/templates/ --dst docs/ --ext md -data api-spec.yaml
```

### Configuration Management

```bash
# Multi-environment configs
templr render -in nginx.conf.tpl -data configs/dev.yaml -out /etc/nginx/nginx.conf
templr render -in nginx.conf.tpl -data configs/prod.yaml -out /etc/nginx/nginx.conf
```

**[More Examples →](docs/examples.md)**

## CI/CD Integration

### GitHub Actions

```yaml
- name: Install templr
  run: curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash

- name: Validate templates
  run: templr lint --src templates/ -d values.yaml --format github-actions --fail-on-warn

- name: Render manifests
  run: templr walk --src templates/ --dst output/ -data values.yaml
```

**[Complete CI/CD Examples →](docs/examples.md#cicd-integration)**

## Exit Codes

templr uses specific exit codes for CI/CD integration:

| Code | Description |
|------|-------------|
| `0` | Success |
| `1` | General error |
| `2` | Template error |
| `3` | Data error |
| `4` | Strict mode error |
| `5` | Guard skipped |
| `6` | Lint warnings (with `--fail-on-warn`) |
| `7` | Lint errors |

**[Complete Exit Code Reference →](docs/cli-reference.md#exit-codes)**

## Contributing

We welcome contributions! To get started:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Support

- **Documentation**: [docs/](docs/README.md)
- **Issues**: [GitHub Issues](https://github.com/kanopi/templr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kanopi/templr/discussions)

## License

This project is licensed under the [MIT License](LICENSE).

## Acknowledgments

- Inspired by [Helm](https://helm.sh/) and Go's `text/template`
- Uses [Sprig](https://masterminds.github.io/sprig/) for template functions
- Built with [Cobra](https://github.com/spf13/cobra) for CLI

---

**[📖 Read the Full Documentation](docs/README.md)** | **[🚀 View Examples](docs/examples.md)** | **[⚙️ Configuration Guide](docs/configuration.md)**
