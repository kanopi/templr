# Configuration Files

This guide covers how to use `.templr.yaml` configuration files to manage templr settings across your projects and teams.

## Table of Contents

- [Overview](#overview)
- [Configuration File Locations](#configuration-file-locations)
- [Basic Configuration](#basic-configuration)
- [Configuration Options Reference](#configuration-options-reference)
- [Configuration Use Cases](#configuration-use-cases)
- [Advanced Scenarios](#advanced-scenarios)
- [Project Structure Best Practices](#project-structure-best-practices)

## Overview

templr supports configuration files to:

- **Set defaults** - Avoid repetitive command-line flags
- **Enforce policies** - Security and validation rules
- **Team consistency** - Same settings for all developers
- **Environment management** - Different configs for dev/prod
- **CI/CD integration** - Automated validation and deployment

## Configuration File Locations

templr automatically loads configuration from these locations (in order of precedence):

1. **Specified config** via `--config` flag (highest priority)
2. **Project config**: `.templr.yaml` in the current directory
3. **User config**: `~/.config/templr/config.yaml`
4. **Built-in defaults** (lowest priority)

**Important**: CLI flags always override configuration file settings.

### Precedence Example

```
CLI flags > --config file > .templr.yaml > ~/.config/templr/config.yaml > defaults
```

## Basic Configuration

Create a `.templr.yaml` file in your project root:

```yaml
# File handling
files:
  extensions:
    - tpl
    - md      # Treat .md files as templates
    - yaml    # Treat .yaml files as templates

  default_values_file: ./values.yaml
  default_templates_dir: ./templates

# Template engine
template:
  left_delimiter: "{{"
  right_delimiter: "}}"
  default_missing: "<no value>"

# Linting rules
lint:
  # Fail CI builds on warnings
  fail_on_warn: true

  # Treat undefined variables as errors (not warnings)
  fail_on_undefined: true

  # Enable strict mode by default
  strict_mode: true

  # Files to exclude from linting
  exclude:
    - "_*.tpl"          # Helper templates
    - "**/test/**"      # Test fixtures
    - "**/*.backup.*"   # Backup files

  # Block dangerous functions
  disallow_functions:
    - env               # No environment variable access
    - exec              # No command execution

  # Require these variables in all templates
  required_vars:
    - name
    - version
    - environment

# Rendering defaults
render:
  dry_run: false
  inject_guard: true
  guard_string: "#templr generated"
  prune_empty_dirs: true

# Output formatting
output:
  color: auto           # auto, always, never
  verbose: false
```

## Configuration Options Reference

### Files Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `extensions` | array | Additional file extensions to treat as templates | `["tpl"]` |
| `default_templates_dir` | string | Default templates directory | `./templates` |
| `default_output_dir` | string | Default output directory | `./out` |
| `default_values_file` | string | Default values file path | `./values.yaml` |
| `helpers` | array | Helper template patterns | `["_helpers*.tpl"]` |

### Template Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `left_delimiter` | string | Left template delimiter | `{{` |
| `right_delimiter` | string | Right template delimiter | `}}` |
| `default_missing` | string | String to render for missing values | `<no value>` |

### Lint Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `fail_on_warn` | bool | Exit with error code on warnings | `false` |
| `fail_on_undefined` | bool | Treat undefined variables as errors | `false` |
| `strict_mode` | bool | Enable strict mode by default | `false` |
| `output_format` | string | Default output format (text, json, github-actions) | `text` |
| `exclude` | array | File patterns to exclude from linting | `[]` |
| `disallow_functions` | array | Template functions to block | `[]` |
| `required_vars` | array | Variables that must be present | `[]` |
| `no_undefined_check` | bool | Skip undefined variable checking | `false` |

### Render Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `dry_run` | bool | Preview changes without writing | `false` |
| `inject_guard` | bool | Auto-inject guard comment | `true` |
| `guard_string` | string | Guard string for overwrite protection | `#templr generated` |
| `prune_empty_dirs` | bool | Remove empty directories after rendering | `true` |

### Output Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `color` | string | Color output (auto, always, never) | `auto` |
| `verbose` | bool | Verbose output | `false` |
| `quiet` | bool | Minimal output | `false` |

## Configuration Use Cases

### Security-Focused Project

Block access to dangerous functions:

```yaml
lint:
  disallow_functions:
    - env                # No environment access
    - exec               # No command execution
    - getHostByName      # No DNS lookups
  fail_on_warn: true
  strict_mode: true
```

### Kubernetes Templates

```yaml
files:
  extensions: [yaml, tpl]
  default_templates_dir: ./k8s/templates
  default_values_file: ./k8s/values.yaml

lint:
  required_vars:
    - namespace
    - image
    - replicas
  fail_on_undefined: true
```

### Documentation Generator

```yaml
files:
  extensions: [md, tpl, txt]
  default_templates_dir: ./docs/templates

template:
  left_delimiter: "<%"
  right_delimiter: "%>"

lint:
  fail_on_undefined: false  # Docs may have optional variables
```

### CI/CD Pipeline

```yaml
lint:
  fail_on_warn: true
  fail_on_undefined: true
  output_format: github-actions

output:
  color: never
```

## Advanced Scenarios

### Team Workflow with User and Project Configs

Allow individual developers to customize their experience while maintaining project standards:

**Project config** (`.templr.yaml` - committed to git):
```yaml
# Project-wide standards
lint:
  required_vars:
    - appName
    - environment
  disallow_functions:
    - env
  fail_on_undefined: true

files:
  extensions: [yaml, tpl]
  default_templates_dir: ./templates
```

**User config** (`~/.config/templr/config.yaml` - personal preferences):
```yaml
# Personal preferences (don't override project security rules)
output:
  color: always
  verbose: true

render:
  dry_run: true  # Preview by default, override with --dry-run=false
```

The project config enforces security and standards, while the user config allows personal workflow customization.

### Multi-Environment Setup

Manage different environments with separate config files:

```bash
# Development environment
cat > .templr.dev.yaml <<EOF
files:
  default_values_file: ./values.dev.yaml

lint:
  fail_on_undefined: false  # Allow undefined vars in dev
  fail_on_warn: false

output:
  verbose: true
EOF

# Production environment
cat > .templr.prod.yaml <<EOF
files:
  default_values_file: ./values.prod.yaml

lint:
  fail_on_undefined: true   # Strict in production
  fail_on_warn: true
  strict_mode: true

render:
  dry_run: false
  inject_guard: true
EOF

# Use with --config flag
templr lint --config .templr.prod.yaml --src ./templates
```

### Configuration Precedence Example

Understanding how settings override each other:

**User config** (`~/.config/templr/config.yaml`):
```yaml
lint:
  fail_on_warn: false        # Set to false
  output_format: text
```

**Project config** (`.templr.yaml`):
```yaml
lint:
  fail_on_warn: true         # Overrides user config
  fail_on_undefined: true
```

**CLI flags** (highest priority):
```bash
# This will use fail_on_warn: false (CLI overrides all configs)
templr lint --src ./templates --fail-on-warn=false
```

**Result**:
- `fail_on_warn=false` (CLI wins)
- `fail_on_undefined=true` (from project config)
- `output_format=text` (from user config)

### Migration from CLI Flags to Config File

Convert existing command-line workflows to configuration files:

**Before** (command-line flags):
```bash
templr lint \
  --src ./templates \
  -d ./values.yaml \
  --fail-on-warn \
  --fail-on-undefined \
  --format json \
  --strict \
  --ext md \
  --ext yaml
```

**After** (`.templr.yaml`):
```yaml
files:
  extensions: [tpl, md, yaml]
  default_values_file: ./values.yaml
  default_templates_dir: ./templates

lint:
  fail_on_warn: true
  fail_on_undefined: true
  strict_mode: true
  output_format: json
```

**New command** (much simpler):
```bash
templr lint --src ./templates
```

**Benefits**:
- Commands are shorter and clearer
- Settings are documented and version-controlled
- Consistent across team members
- Easy to audit and update

### Monorepo with Multiple Projects

Different templating rules for different parts of a monorepo:

```
monorepo/
├── .templr.yaml                    # Root defaults
├── services/
│   └── api/
│       ├── .templr.yaml           # API-specific config
│       └── templates/
├── infrastructure/
│   └── kubernetes/
│       ├── .templr.yaml           # K8s-specific config
│       └── templates/
└── docs/
    ├── .templr.yaml               # Docs-specific config
    └── templates/
```

**Root `.templr.yaml`** (base defaults):
```yaml
output:
  color: auto

lint:
  fail_on_warn: true
```

**services/api/.templr.yaml** (API overrides):
```yaml
files:
  extensions: [yaml, json, tpl]

lint:
  required_vars:
    - serviceName
    - apiVersion
  disallow_functions:
    - env
```

**infrastructure/kubernetes/.templr.yaml** (K8s overrides):
```yaml
files:
  extensions: [yaml, tpl]

lint:
  required_vars:
    - namespace
    - image
    - replicas
  fail_on_undefined: true
```

**docs/.templr.yaml** (docs overrides):
```yaml
files:
  extensions: [md, tpl]

template:
  left_delimiter: "[["
  right_delimiter: "]]"

lint:
  fail_on_undefined: false  # Docs can have optional vars
```

Each directory uses the appropriate config when you run templr from that location.

## Project Structure Best Practices

### Recommended Layout

```
myproject/
├── .templr.yaml              # Project configuration
├── values.yaml               # Default values
├── values.dev.yaml           # Development overrides
├── values.prod.yaml          # Production overrides
├── templates/
│   ├── _helpers.tpl          # Shared helper templates
│   ├── config.yaml.tpl
│   ├── deployment.yaml.tpl
│   └── service.yaml.tpl
└── output/                   # Generated files
```

### Helper Templates Organization

Use underscore prefix for helper templates that shouldn't be rendered directly:

```yaml
# .templr.yaml
lint:
  exclude:
    - "_*.tpl"                # Don't lint helper templates
    - "**/test/**"            # Don't lint test fixtures
```

### Version Control

**Commit to git**:
- `.templr.yaml` - Project standards
- `.templr.dev.yaml` - Development config
- `.templr.prod.yaml` - Production config
- `.templr.yaml.example` - Example/documentation

**Don't commit**:
- `~/.config/templr/config.yaml` - Personal preferences
- Local overrides or secrets

### Security Best Practices

```yaml
lint:
  # Block dangerous functions
  disallow_functions:
    - env                     # Environment variables
    - exec                    # Command execution
    - getHostByName           # DNS lookups

  # Enforce required security variables
  required_vars:
    - environment
    - version
    - securityContext

  # Strict validation
  fail_on_warn: true
  fail_on_undefined: true
  strict_mode: true
```

## Tips and Best Practices

### Start Simple

Begin with basic configuration and add rules as needed:

```yaml
# Minimal starting config
files:
  default_values_file: ./values.yaml

lint:
  fail_on_warn: true
```

### Use Comments

Document your configuration decisions:

```yaml
lint:
  # We allow env() in dev but block it in production
  disallow_functions:
    - exec              # Never allow command execution

  # Required for audit compliance
  required_vars:
    - owner
    - costCenter
```

### Test Configurations

Validate your config before committing:

```bash
# Test with --config flag
templr lint --config .templr.yaml --src ./templates

# Test precedence
templr lint --config .templr.yaml --src ./templates --fail-on-warn=false
```

### Share Examples

Include a `.templr.yaml.example` file in your repository:

```yaml
# .templr.yaml.example
# Copy this to .templr.yaml and customize for your environment

files:
  default_values_file: ./values.yaml

lint:
  fail_on_warn: true
  required_vars:
    - projectName
    - environment
```

## Troubleshooting

### Config Not Loading

**Problem**: Configuration seems ignored

**Check**:
1. Verify file name: `.templr.yaml` (not `.templr.yml`)
2. Check YAML syntax: `yamllint .templr.yaml`
3. Verify location: Current directory or `~/.config/templr/`
4. Test with `--config` flag explicitly

### Unexpected Precedence

**Problem**: Wrong config is being used

**Debug**:
```bash
# Use verbose mode to see config loading
templr lint --src ./templates --verbose

# Test with explicit config
templr lint --config .templr.yaml --src ./templates
```

### YAML Syntax Errors

**Problem**: "error parsing config"

**Solution**:
- Validate YAML syntax online or with `yamllint`
- Check indentation (use spaces, not tabs)
- Ensure arrays use proper syntax: `[item1, item2]` or list format
- Quote special characters: `"$VAR"`, `"*.tpl"`

## Next Steps

- [Templating Guide](templating-guide.md) - Learn template syntax
- [CLI Reference](cli-reference.md) - All commands and flags
- [Examples](examples.md) - Real-world use cases
- [Back to Documentation Hub](README.md)
