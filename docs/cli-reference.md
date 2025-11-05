# CLI Reference

Complete command-line reference for templr.

## Table of Contents

- [Commands](#commands)
- [Global Flags](#global-flags)
- [Exit Codes](#exit-codes)
- [Environment Variables](#environment-variables)
- [Stdin/Stdout](#stdinstdout)

## Commands

### `templr render`

Render a single template file to an output file or stdout.

**Syntax:**
```bash
templr render [flags]
```

**Flags:**
- `-i, --in <file>` - Template file (omit for stdin)
- `-o, --out <file>` - Output file (omit for stdout)
- `--helpers <pattern>` - Glob pattern for helper templates (default: `_helpers*.tpl`)

**Examples:**
```bash
# Render from file to file
templr render -in template.tpl -data values.yaml -out output.txt

# Render from stdin to stdout
echo 'Hello {{ .name }}' | templr render -data values.yaml

# Render with --set overrides
templr render -in template.tpl --set name=World -out output.txt

# Disable helper loading
templr render -in template.tpl -data values.yaml --helpers=""
```

**See also:** [Examples - Single File Rendering](examples.md#single-file-rendering)

---

### `templr dir`

Render templates from a directory with shared helpers and partials.

**Syntax:**
```bash
templr dir --dir <path> [flags]
```

**Flags:**
- `--dir <path>` - Directory containing templates (required)
- `-i, --in <name>` - Entry template name (default: 'root' or first template)
- `-o, --out <file>` - Output file (omit for stdout)

**Examples:**
```bash
# Render using an entry template
templr dir --dir templates/ -in main.tpl -data values.yaml -out output.txt

# Render with auto-detected entry (looks for "root" template)
templr dir --dir templates/ -data values.yaml -out output.txt
```

**See also:** [Examples - Directory Mode](examples.md#directory-mode)

---

### `templr walk`

Recursively walk through a directory and render all templates, mirroring the directory structure.

**Syntax:**
```bash
templr walk --src <path> --dst <path> [flags]
```

**Flags:**
- `--src <path>` - Source template directory (required)
- `--dst <path>` - Destination output directory (required)

**Examples:**
```bash
# Walk and render all templates
templr walk --src templates/ --dst output/

# Walk with additional file extensions
templr walk --src templates/ --dst output/ --ext md --ext txt

# Dry-run to preview changes
templr walk --src templates/ --dst output/ --dry-run
```

**Behavior:**
- Template file extensions (`.tpl` and any specified with `--ext`) are stripped from output filenames
- Directory structure is preserved
- Empty directories are automatically pruned (unless `--prune-empty-dirs=false`)

**See also:** [Examples - Walk Mode](examples.md#walk-mode)

---

### `templr lint`

Validate template syntax and detect issues without rendering.

**Syntax:**
```bash
templr lint [flags]
```

**Flags:**
- `-i, --in <file>` - Single template file to lint
- `--dir <path>` - Directory of templates to lint
- `--src <path>` - Source directory tree to walk and lint
- `--fail-on-warn` - Exit with error code on warnings (default: errors only)
- `--format <format>` - Output format: `text`, `json`, `github-actions` (default: `text`)
- `--no-undefined-check` - Skip undefined variable detection

**Examples:**
```bash
# Lint a single template file
templr lint -i template.tpl -d values.yaml

# Lint all templates in a directory
templr lint --dir templates/ -d values.yaml

# Lint entire directory tree
templr lint --src templates/ -d values.yaml

# Fail CI on warnings (not just errors)
templr lint --src templates/ -d values.yaml --fail-on-warn

# Skip undefined variable checking (syntax only)
templr lint --src templates/ --no-undefined-check

# Output in JSON format for programmatic use
templr lint --src templates/ -d values.yaml --format json

# GitHub Actions format for annotations
templr lint --src templates/ -d values.yaml --format github-actions
```

**Checks performed:**
- Template syntax correctness (parse errors, missing `{{end}}`, etc.)
- Undefined variable references (when data is provided)
- Disallowed function usage (when configured)
- Required variable presence (when configured)

**Exit codes:**
- `0` - No issues found
- `6` - Warnings found (with `--fail-on-warn`)
- `7` - Errors found

**See also:** [Examples - Linting](examples.md#linting-and-validation)

---

### `templr version`

Print version information.

**Syntax:**
```bash
templr version
```

**Examples:**
```bash
templr version
# Output: 1.0.0
```

**Legacy syntax:**
```bash
templr --version
templr -version
```

---

## Global Flags

These flags are available for all commands:

### Data and Values

| Flag | Description | Default |
|------|-------------|---------|
| `-d, --data <file>` | Path to base JSON or YAML data file | - |
| `-f <file>` | Additional values files (YAML/JSON). Repeatable. | - |
| `--set <key=value>` | Key=value overrides. Repeatable. Supports dotted keys. | - |

**Examples:**
```bash
# Load base data
templr render -in template.tpl -data values.yaml

# Load multiple data files
templr render -in template.tpl -data values.yaml -f env.yaml -f secrets.yaml

# Set individual values
templr render -in template.tpl --set name=myapp --set version=1.0.0

# Set nested values with dot notation
templr render -in template.tpl --set app.name=myapp --set app.version=1.0.0

# Combine all methods (precedence: --set > -f > -d)
templr render -in template.tpl -data values.yaml -f prod.yaml --set replicas=5
```

### Template Engine

| Flag | Description | Default |
|------|-------------|---------|
| `--ldelim <string>` | Left delimiter | `{{` |
| `--rdelim <string>` | Right delimiter | `}}` |
| `--default-missing <string>` | String to render when a variable/key is missing | `<no value>` |
| `--strict` | Fail on missing keys | `false` |

**Examples:**
```bash
# Use custom delimiters (e.g., to avoid conflicts with other template systems)
templr render -in template.tpl -data values.yaml --ldelim "[["  --rdelim "]]"

# Change default missing value
templr render -in template.tpl -data values.yaml --default-missing "N/A"

# Enable strict mode (fail on undefined variables)
templr render -in template.tpl -data values.yaml --strict
```

### File Extensions

| Flag | Description | Default |
|------|-------------|---------|
| `--ext <extension>` | Additional template file extensions (e.g., md, txt). Repeatable. Omit the leading dot. | `tpl` only |

**Examples:**
```bash
# Process .md files as templates
templr walk --src templates/ --dst output/ --ext md

# Process multiple extensions
templr walk --src templates/ --dst output/ --ext md --ext txt --ext yaml
```

### Guards and Overwrite Protection

| Flag | Description | Default |
|------|-------------|---------|
| `--guard <string>` | Guard string required in existing files to allow overwrite | `#templr generated` |
| `--inject-guard` | Automatically insert the guard as a comment into written files | `true` |

**Examples:**
```bash
# Use custom guard string
templr walk --src templates/ --dst output/ --guard "#generated by templr"

# Disable guard injection
templr walk --src templates/ --dst output/ --inject-guard=false
```

**Guard behavior:**
- When writing to an existing file, templr only overwrites if the file contains the guard string
- With `--inject-guard`, templr automatically inserts the guard comment in the correct format for the file type
- Helps prevent accidental overwrites of manually edited files

### Execution Modes

| Flag | Description | Default |
|------|-------------|---------|
| `--dry-run` | Preview which files would be rendered (no writes) | `false` |

**Examples:**
```bash
# Preview changes without writing
templr walk --src templates/ --dst output/ --dry-run
```

### Output Control

| Flag | Description | Default |
|------|-------------|---------|
| `--no-color` | Disable colored output (useful for CI/non-ANSI terminals) | `false` |
| `-v, --verbose` | Verbose output | `false` |
| `-q, --quiet` | Minimal output | `false` |

**Examples:**
```bash
# Disable colors for CI
templr lint --src templates/ -d values.yaml --no-color

# Verbose output for debugging
templr walk --src templates/ --dst output/ --verbose
```

### Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `--config <file>` | Path to config file | `.templr.yaml` or `~/.config/templr/config.yaml` |

**Examples:**
```bash
# Use specific config file
templr lint --config .templr.prod.yaml --src templates/

# Skip config loading (use only CLI flags)
templr render -in template.tpl -data values.yaml --config ""
```

---

## Exit Codes

templr uses specific exit codes for CI/CD integration:

| Code | Name | Description |
|------|------|-------------|
| `0` | `ExitOK` | Success - no issues found |
| `1` | `ExitGeneral` | General error (invalid arguments, unknown error) |
| `2` | `ExitTemplateError` | Template parsing or rendering error |
| `3` | `ExitDataError` | Data loading error (invalid YAML/JSON, file not found) |
| `4` | `ExitStrictError` | Strict mode error (missing required variable) |
| `5` | `ExitGuardSkipped` | File skipped due to missing guard string |
| `6` | `ExitLintWarn` | Lint warnings found (with `--fail-on-warn`) |
| `7` | `ExitLintError` | Lint errors found |

**CI/CD Usage:**
```bash
# GitHub Actions example
- name: Lint templates
  run: templr lint --src templates/ -d values.yaml
  # Fails workflow if exit code != 0

# Check specific exit codes in scripts
templr walk --src templates/ --dst output/
EXIT_CODE=$?
if [ $EXIT_CODE -eq 5 ]; then
  echo "Files were skipped due to guard protection"
elif [ $EXIT_CODE -ne 0 ]; then
  echo "Error occurred: $EXIT_CODE"
  exit $EXIT_CODE
fi
```

---

## Environment Variables

Currently, templr does not use environment variables for configuration. All settings must be provided via:
- CLI flags
- Configuration files (`.templr.yaml`)
- User config (`~/.config/templr/config.yaml`)

**Note:** While you can use `--set` with environment variables in shell, templr itself doesn't read env vars:
```bash
# Shell expands $VERSION before passing to templr
templr render -in template.tpl --set version=$VERSION
```

---

## Stdin/Stdout

### Reading from Stdin

If `-in` is not provided, templr reads the template from standard input:

```bash
# Pipe template content
echo 'Hello {{ .name }}!' | templr render --set name=World

# Here-doc
templr render --set name=World <<EOF
Hello {{ .name }}!
Version: {{ .version }}
EOF

# From a command
cat template.tpl | templr render -data values.yaml
```

### Writing to Stdout

If `-out` is not provided, rendered output is written to standard output:

```bash
# Capture output
OUTPUT=$(templr render -in template.tpl -data values.yaml)

# Pipe to other commands
templr render -in template.tpl -data values.yaml | grep "version"

# Redirect to file
templr render -in template.tpl -data values.yaml > output.txt
```

### Pipeline Usage

```bash
# Complete pipeline
cat template.tpl | templr render --set name=myapp | tee output.txt | head -5

# With data files
echo '{{ .name }}-{{ .env }}' | templr render -data values.yaml --set env=prod

# Multiple stages
templr render -in stage1.tpl -data values.yaml | \
  templr render --set stage=2 > final.txt
```

---

## Legacy Syntax

For backward compatibility, templr supports legacy flag-based syntax:

```bash
# Legacy single-file
templr -in template.tpl -data values.yaml -out output.txt

# Legacy directory mode
templr --dir templates/ -in main.tpl -data values.yaml -out output.txt

# Legacy walk mode
templr --walk --src templates/ --dst output/

# Legacy version
templr -version
```

**Note:** New subcommand syntax is recommended for clarity. Legacy syntax may be deprecated in future versions.

---

## Tips and Best Practices

### Use Configuration Files

Instead of long command lines, use `.templr.yaml`:

```yaml
files:
  default_values_file: ./values.yaml
  extensions: [tpl, md, yaml]

lint:
  fail_on_warn: true
  fail_on_undefined: true
```

Then simply:
```bash
templr lint --src templates/
```

See: [Configuration Guide](configuration.md)

### Lint Before Rendering

Always validate templates before rendering in production:

```bash
#!/bin/bash
# Deploy script
set -e  # Exit on error

# Validate
templr lint --src templates/ -d values.prod.yaml --fail-on-warn

# Render
templr walk --src templates/ --dst output/ -data values.prod.yaml
```

### Use Dry-Run

Preview changes before writing:

```bash
# See what would be rendered
templr walk --src templates/ --dst output/ --dry-run

# Then actually render
templr walk --src templates/ --dst output/
```

### Combine with Version Control

```bash
# Render and commit
templr walk --src templates/ --dst manifests/ -data values.yaml
git add manifests/
git commit -m "Update manifests"
```

### CI/CD Integration

```yaml
# .github/workflows/lint.yml
name: Lint Templates
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install templr
        run: |
          curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/get-templr.sh | bash
      - name: Lint templates
        run: |
          templr lint \
            --src templates/ \
            -d values.yaml \
            --format github-actions \
            --fail-on-warn
```

---

## Next Steps

- [Configuration Guide](configuration.md) - Set up `.templr.yaml`
- [Templating Guide](templating-guide.md) - Learn template syntax
- [Examples](examples.md) - Real-world use cases
- [Back to Documentation Hub](README.md)
