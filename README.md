# templr

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/kanopi/templr/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/kanopi/templr/tree/main) [![Docker Pulls](https://img.shields.io/docker/pulls/kanopi/templr)](https://hub.docker.com/r/kanopi/templr) [![Latest Release](https://img.shields.io/github/v/release/kanopi/templr)](https://github.com/kanopi/templr/releases) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)


## Overview

templr is a Go-based templating CLI inspired by Helm and Go's `text/template` package. It allows you to render templates from single files or entire directories, providing powerful features to manage complex templating workflows. templr is designed to be flexible and easy to use, making it ideal for generating configuration files, manifests, or any text-based output from templates.

## Features

- **Multi-file rendering**: Render single template files or entire directories of templates.
- **Walk mode**: Recursively walk through directories and render all templates found.
- **Lint mode**: Validate template syntax and detect undefined variables without rendering.
- **`.Files` API**: Access files within the template directory during rendering.
- **Strict mode**: Enforce strict template parsing and execution to catch errors early.
- **Guards**: Use the `--guard` flag to conditionally skip rendering files based on template output.
- **Dry-run**: Preview rendered output without writing files to disk using the `--dry-run` flag.
- **Pruning empty directories**: Automatically detect and prune directories containing only whitespace or empty output.
- **Flexible data input**: Pass data via `--set` flags or load from JSON/YAML files with `--data`.
- **Custom extensions**: Use the `--ext` flag to include additional template file extensions (e.g., md, txt). `.tpl` is always included by default.
- **CI/CD friendly**: Exit codes, JSON output, and GitHub Actions integration for automated workflows.

## Installation

### Download Latest Release

Download the latest pre-built binary for your platform from the [GitHub Releases](https://github.com/kanopi/templr/releases) page. Extract the archive and place the `templr` binary in your system PATH.

### Install via Hombrew

You can install templr using Homebrew by first tapping the repository and then installing the package:

```bash
brew tap kanopi/templr
brew install templr
```

### Install via curl

templr can be installed using a one-line command that downloads and installs the latest version automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash
```

To install a specific version, specify the tag as an argument:

```bash
curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash -s 1.2.3
```

### Using Docker

You can run templr using the official Docker image without installing anything locally:

```bash
docker run --rm -v $(pwd):/work -w /work kanopi/templr --walk --src /work/templates --dst /work/out
```

Or to run a single template file:

```bash
docker run --rm -v $(pwd):/work -w /work kanopi/templr -in /work/template.tpl -data /work/values.yaml -out /work/output.yaml
```

## Usage and Scenarios

templr supports rendering templates in various modes and includes a full suite of examples to help you get started and verify functionality.

### Rendering Modes

- **Single-file mode** (`render`): Render a single template file.

  ```bash
  # New subcommand syntax (recommended)
  templr render -in path/to/template.tpl -data values.yaml -out output.txt

  # Legacy syntax (still supported)
  templr -in path/to/template.tpl -data values.yaml -out output.txt
  ```

- **Directory mode** (`dir`): Render all templates in a directory.

  ```bash
  # New subcommand syntax (recommended)
  templr dir --dir path/to/templates/ -in main.tpl -data values.yaml -out out.txt

  # Legacy syntax (still supported)
  templr --dir path/to/templates/ -in main.tpl -data values.yaml -out out.txt
  ```

- **Walk mode** (`walk`): Recursively walk through a directory and render all templates.

  ```bash
  # New subcommand syntax (recommended)
  templr walk --src path/to/templates/ --dst path/to/output/
  templr walk --src path/to/templates/ --dst path/to/output/ --ext md --ext txt

  # Legacy syntax (still supported)
  templr --walk --src path/to/templates/ --dst path/to/output/
  templr --walk --src path/to/templates/ --dst path/to/output/ --ext md --ext txt
  ```

- **Lint mode** (`lint`): Validate template syntax without rendering. Detect parse errors and undefined variables.

  ```bash
  # Lint a single template file
  templr lint -i template.tpl -d values.yaml

  # Lint all templates in a directory
  templr lint --dir path/to/templates/ -d values.yaml

  # Lint entire directory tree
  templr lint --src path/to/templates/ -d values.yaml

  # Fail CI pipeline on warnings (not just errors)
  templr lint --src path/to/templates/ -d values.yaml --fail-on-warn

  # Skip undefined variable checking (syntax only)
  templr lint --src path/to/templates/ --no-undefined-check

  # Output in JSON format for programmatic use
  templr lint --src path/to/templates/ -d values.yaml --format json

  # GitHub Actions format for annotations
  templr lint --src path/to/templates/ -d values.yaml --format github-actions
  ```

  Lint mode helps catch template errors early in CI/CD pipelines by:
  - Checking template syntax correctness (parse errors, missing `{{end}}`, etc.)
  - Detecting undefined variable references when data is provided
  - Reporting issues with file paths and line numbers
  - Supporting multiple output formats (text, json, github-actions)

> **Note**: The new subcommand syntax is recommended for clarity. The legacy flag-based syntax is maintained for backward compatibility.

### Custom Template Extensions

By default, templr processes files ending in `.tpl`. You can extend this behavior with the `--ext` flag to include additional text-based extensions such as `md`, `txt`, `html`, etc. This allows you to use templr for Markdown, documentation, or configuration file templating.

### Using stdin and stdout

templr can also read templates from **stdin** and write output to **stdout**.

If `-in` is not provided, templr reads the template from standard input.
If `-out` is not provided, the rendered output is written to standard output.

This enables templr to be used easily in pipelines or shell scripts.

#### Examples

```bash
# Render from stdin (new syntax)
echo 'Hello {{ .name }}' | templr render -data values.yaml

# Render from stdin (legacy syntax)
echo 'Hello {{ .name }}' | templr -data values.yaml

# Render to stdout
templr render -in template.tpl -data values.yaml > output.txt
```

This feature is especially useful when integrating templr into automated workflows or CI/CD pipelines.

## Configuration File

templr supports configuration files to set defaults and enforce policies across your project or for your user account. This is especially useful for:

- Setting up lint rules for CI/CD
- Enforcing security policies (disallowing dangerous functions)
- Defining project-specific file extensions and paths
- Maintaining consistent settings across team members

### Configuration File Locations

templr automatically loads configuration from these locations (in order of precedence):

1. **Specified config** via `--config` flag (highest priority)
2. **Project config**: `.templr.yaml` in the current directory
3. **User config**: `~/.config/templr/config.yaml`
4. **Built-in defaults** (lowest priority)

CLI flags always override configuration file settings.

### Example Configuration

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

### Configuration Options Reference

#### Files Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `extensions` | array | Additional file extensions to treat as templates | `["tpl"]` |
| `default_templates_dir` | string | Default templates directory | `./templates` |
| `default_output_dir` | string | Default output directory | `./out` |
| `default_values_file` | string | Default values file path | `./values.yaml` |
| `helpers` | array | Helper template patterns | `["_helpers*.tpl"]` |

#### Template Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `left_delimiter` | string | Left template delimiter | `{{` |
| `right_delimiter` | string | Right template delimiter | `}}` |
| `default_missing` | string | String to render for missing values | `<no value>` |

#### Lint Configuration

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

#### Render Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `dry_run` | bool | Preview changes without writing | `false` |
| `inject_guard` | bool | Auto-inject guard comment | `true` |
| `guard_string` | string | Guard string for overwrite protection | `#templr generated` |
| `prune_empty_dirs` | bool | Remove empty directories after rendering | `true` |

#### Output Configuration

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `color` | string | Color output (auto, always, never) | `auto` |
| `verbose` | bool | Verbose output | `false` |
| `quiet` | bool | Minimal output | `false` |

### Configuration Use Cases

#### Security-Focused Project

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

#### Kubernetes Templates

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

#### Documentation Generator

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

#### CI/CD Pipeline

```yaml
lint:
  fail_on_warn: true
  fail_on_undefined: true
  output_format: github-actions

output:
  color: never
```

### Advanced Configuration Scenarios

#### Team Workflow with User and Project Configs

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

#### Multi-Environment Setup

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

#### Configuration Precedence Example

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

**Result**: `fail_on_warn=false` (CLI wins), `fail_on_undefined=true` (from project config), `output_format=text` (from user config).

#### Migration from CLI Flags to Config File

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

Benefits:
- Commands are shorter and clearer
- Settings are documented and version-controlled
- Consistent across team members
- Easy to audit and update

#### Monorepo with Multiple Projects

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

### Common Command-line Flags

- `-in`: A single template file (single-file mode) or an entry template name when used with `--dir`.
- `--dir`: Directory containing templates for multi-file rendering.
- `--src`: Source directory for templates when using `--walk` mode. templr will recursively search this directory for template files.
- `--dst`: Destination directory where rendered templates will be written when using `--walk` mode.
- `--walk`: Enable recursive walk mode for directory templates.
- `--set key=value`: Set template data key-value pairs.
- `--data path/to/data.yaml`: Load template data from a JSON or YAML file.
- `--strict`: Enable strict mode for template parsing and execution.
- `--guard`: Enable guard behavior to conditionally skip rendering files.
- `--dry-run`: Render templates without writing output to disk.
- `--helpers`: Specify a glob pattern for helper templates (default: `_helpers*.tpl`). Set to an empty string to skip loading helpers in single-file mode.
- `--ext`: Specify additional template file extensions to treat as templates (e.g., md, txt). Repeatable; omit the leading dot.
- `--version`: Display the current version and exit.

### 🧩 Additional Flags and Helpers

| Flag / Helper | Description | Default |
|----------------|-------------|----------|
| `--default-missing` | String to render when a variable/key is missing (works with `missingkey=default`). | `<no value>` |
| `safe` (template helper) | Template function usable inside templates: `{{ safe .var "fallback" }}` — renders a fallback when the variable is missing or empty. | N/A |

#### Example Usage

```bash
# Render with a custom placeholder for missing values
templr --in template.tpl --out output.txt --default-missing "N/A"

# Example using the safe helper
# template.tpl:
# Name: {{ safe .user.name "anonymous" }}
# Output:
# Name: anonymous
```

💡 **Tip:**
You can combine both behaviors — setting `--default-missing` for global fallback values while still using `safe` inside templates for specific variables.


### Versioning

templr includes a built-in version command to display the current version of the binary.

```bash
# New subcommand syntax
templr version

# Legacy syntax (still supported)
templr -version
```

The version is determined in this order:
1. From build-time flags provided by CircleCI (`-ldflags "-X main.Version=<tag>"`).
2. Defaults to `dev` when not provided.

This allows templr builds to display accurate version information when built from branches or tags.

### Examples & Testing

templr includes a full suite of ready-made examples and integration tests to help you learn and verify functionality.

#### Running Common Example Cases

- **Single-file rendering**

  ```bash
  templr -in template.tpl -data values.yaml -out output.yaml
  ```

- **Walk mode**

  ```bash
  templr --walk -src path/to/templates/ -dst path/to/output/
  ```

- **Guard behavior**

  Use pre-existing files containing or lacking the `#templr generated` marker to see how guard mode conditionally skips rendering.

  The `--guard` flag controls the overwrite behavior of templr by using a marker string to determine whether a file should be overwritten. By default, this marker is `#templr generated`. When enabled, templr will only overwrite files that contain this marker, helping prevent accidental overwrites of manually edited files.

  You can customize the guard string by passing a different value to the `--guard` flag:

  ```bash
  templr --guard "custom marker"
  ```

  Additionally, templr automatically inserts the guard marker into rendered files in the correct comment syntax for each file type when the `--inject-guard` flag is enabled (which is `true` by default). This ensures the guard marker is present without manual intervention.

  If you prefer to disable automatic insertion of the guard marker, you can set:

  ```bash
  templr --inject-guard=false
  ```

- **Dry-run and pruning empty output**

  Use the `--dry-run` flag to preview output without writing files, and observe how empty or whitespace-only outputs are pruned automatically.

#### Example Directories

The examples include directories such as `playground/` which contain test templates demonstrating features like:

- Accessing `.Files` in templates
- Template includes and partials
- Guard behavior and conditional rendering
- Strict mode enforcement

#### Integration Testing

These examples serve as integration tests to verify that all features of templr work together correctly, providing a practical way to learn and validate templr's capabilities.

### Guard Behavior

The `--guard` flag controls **overwrite behavior** using a marker string (default `#templr generated`). When writing to an existing file, templr will only overwrite if the file already contains the marker. With `--inject-guard` (default `true`), templr inserts the marker in the correct comment style when creating or updating files.

### Dry Run

Using the `--dry-run` flag renders templates and outputs the result to stdout or logs without writing any files to disk. This is useful for previewing changes or debugging templates before applying them.

### Skipping Empty Output

templr automatically detects output that contains only whitespace or is empty and prunes such files and their parent directories. This behavior helps keep your output clean by removing unnecessary empty files and directories generated by templates that produce no meaningful content.

## Documentation

For a full reference of templr’s templating syntax, variables, conditionals, functions, and `.Files` API, see the [docs.md](./docs/docs.md) file.

## License

This project is licensed under the [MIT License](./LICENSE). See the LICENSE file for details.
