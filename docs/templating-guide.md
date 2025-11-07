# Templr Templating Guide

## Table of Contents
1. [Variables and Data Access](#1-variables-and-data-access)
2. [Control Flow](#2-control-flow)
   - [Whitespace Control](#whitespace-control)
   - [Conditionals](#conditionals)
   - [Loops](#loops)
3. [The .Files API](#3-the-files-api)
4. [Helpers and Functions](#4-helpers-and-functions)
5. [Data Precedence and Scoping](#5-data-precedence-and-scoping)
   - [Default values.yaml and values.yml Lookup](#default-valuesyaml-and-valuesyml-lookup)
6. [Advanced Capabilities and Sprig Functions](#6-advanced-capabilities-and-sprig-functions)
7. [Additional Template Functions](#7-additional-template-functions)
   - [Humanization Functions](#humanization-functions)
   - [TOML Support](#toml-support)
   - [Path Functions](#path-functions)
   - [Validation Functions](#validation-functions)
8. [Helper Templates and Pre-Render Variables](#8-helper-templates-and-pre-render-variables)
9. [Guards and Safe Access](#9-guards-and-safe-access)
10. [Comments](#10-comments)
11. [Putting It All Together](#11-putting-it-all-together)
12. [Configuration Files and Project Setup](#12-configuration-files-and-project-setup)
13. [Summary](#summary)

---

# Templr Templating Guide

Welcome to the full guide for using **templr**, a powerful and flexible templating engine designed to simplify generating text files from templates. This document covers all the core concepts, syntax, and features you need to master templr.

---

## 1. Variables and Data Access

Templr templates are populated using data passed in as JSON or YAML objects. You can access variables directly by name:

```gotmpl
Hello, {{ .Name }}!
```

Here, `.Name` accesses the `Name` property from the root data object.

### Nested Data

Access nested data using dot notation:

```gotmpl
User email: {{ .User.Email }}
```

If your data is:

```json
{
  "User": {
    "Email": "user@example.com"
  }
}
```

---


## 2. Control Flow

### Whitespace Control

Go templates (and templr) provide control over whitespace using hyphens within delimiters. This determines how spaces and newlines around template actions are handled.

| Syntax | Effect |
|---------|--------|
| `{{ variable }}` | Keeps surrounding whitespace (default). |
| `{{- variable }}` | Trims whitespace **to the left** of the action. |
| `{{ variable -}}` | Trims whitespace **to the right** of the action. |
| `{{- variable -}}` | Trims whitespace on **both sides** of the action. |

#### Example

```gotmpl
Hello,
{{- if .Name }}
  {{ .Name }}
{{- end }}
!
```

If `.Name` is `"Sean"`, this renders as:

```
Hello,Moto!
```

Without the hyphens, it renders as:

```
Hello,
  Moto
!
```

Whitespace control is especially useful when generating Markdown, YAML, or HTML templates, where extra or missing line breaks can change formatting or validity.

### Conditionals

Use `if`, `else if` (using `else if` or `else if`), and `else` blocks:

```gotmpl
{{ if .IsAdmin }}
  Welcome, admin!
{{ else if .IsUser }}
  Welcome, user!
{{ else }}
  Please log in.
{{ end }}
```

### Loops

Iterate over arrays or maps with `range`:

```gotmpl
{{ range .Items }}
  - {{ . }}
{{ end }}
```

Inside a `range`, `.` is the current item.

---

## 3. The `.Files` API

Templr provides access to files in the input directory via the `.Files` object.

### Basic File Operations

**Read Files:**
- `.Files.Get("filename")` - Returns file content as a string
- `.Files.GetBytes("filename")` - Returns file content as bytes

**File Discovery:**
- `.Files.Glob("pattern")` - Returns list of files matching glob pattern
- `.Files.Exists("path")` - Returns true if file or directory exists
- `.Files.ReadDir("path")` - Returns list of files/directories in a directory

**File Metadata:**
- `.Files.Stat("path")` - Returns file metadata (Name, Size, Mode, ModTime, IsDir)
- `.Files.GlobDetails("pattern")` - Returns file metadata for all matching files

### Reading Files Line-by-Line

- `.Files.Lines("filename")` - Returns file content as array of lines
- `.Files.AsLines("filename")` - Alias for Lines()

**Example:**

```gotmpl
{{- range $idx, $line := .Files.Lines "servers.txt" }}
server{{ $idx }}: {{ $line }}
{{- end }}
```

### Encoding Helpers

**Base64 Encoding:**
```gotmpl
# Perfect for Kubernetes Secrets
apiVersion: v1
kind: Secret
data:
  tls.crt: {{ .Files.AsBase64 "certs/tls.crt" }}
  tls.key: {{ .Files.AsBase64 "certs/tls.key" }}
```

**Other Encodings:**
- `.Files.AsHex("file")` - Returns file content as hexadecimal string
- `.Files.AsDataURL("file", "mime/type")` - Returns data URL for embedding in HTML/CSS

**Data URL Example:**
```gotmpl
<!-- Embed image directly in HTML -->
<img src="{{ .Files.AsDataURL "logo.png" "" }}" alt="Logo">
```

### Parsing Structured Files

**JSON:**
```gotmpl
{{- $config := .Files.AsJSON "config.json" }}
App: {{ $config.app.name }}
Version: {{ $config.app.version }}
```

**YAML:**
```gotmpl
{{- $values := .Files.AsYAML "values.yaml" }}
Database: {{ $values.database.host }}:{{ $values.database.port }}
```

### Advanced Examples

**Conditional File Loading:**
```gotmpl
{{- if .Files.Exists "config/prod.yaml" }}
{{- $prodConfig := .Files.AsYAML "config/prod.yaml" }}
# Production configuration loaded
{{- end }}
```

**Directory Listing:**
```gotmpl
Files in configs/:
{{- range .Files.ReadDir "configs" }}
  - {{ . }}
{{- end }}
```

**File Metadata:**
```gotmpl
{{- range .Files.GlobDetails "*.yaml" }}
- {{ .Name }} ({{ .Size }} bytes, modified {{ .ModTime }})
{{- end }}
```

**Glob Pattern Matching:**
```gotmpl
{{ range .Files.Glob("*.txt") }}
- File: {{ . }}
{{ end }}
```

---

## 4. Helpers and Functions

Templr supports built-in helper functions to manipulate data:

- `len`: returns length of a string, array, or map.
- `upper`: converts a string to uppercase.
- `lower`: converts a string to lowercase.
- `default`: provides a default value if the variable is empty.

Example:

```gotmpl
{{ if eq (lower .Role) "admin" }}
  You have administrator privileges.
{{ end }}

Total items: {{ len .Items }}

Hello, {{ default .Name "Guest" }}!
```

---

## 5. Data Precedence and Scoping

Templr templates have a data precedence order:

1. Variables defined within the template (using `{{ $var := ... }}`).
2. Data passed into the template from the CLI or API.
3. Special objects like `.Files`.

Variables defined within the template are scoped to their block but can be passed down by assignment.

Example:

```gotmpl
{{ $greeting := "Hello" }}
{{ range .Users }}
  {{ $greeting }}, {{ .Name }}!
{{ end }}
```

---

### Default values.yaml and values.yml Lookup

templr automatically looks for a default `values.yaml` or `values.yml` file in the template root directory.

- **Single-file mode**: Looks in the same directory as the input file.
- **Directory mode**: Looks in the directory specified with `-dir`.
- **Walk mode**: Looks in the root of the source directory (`-src`).

If found, this default values file is merged **before** any explicitly provided `--data` or `-f` files and `--set` overrides.  
The final merge order is:

1. Default values.yaml (if present)
2. Data file from `--data`
3. Overlay files from `-f`
4. Inline values from `--set`

This behavior mimics Helm’s automatic values merging, allowing you to define sensible defaults per template set.

---

## 6. Advanced Capabilities and Sprig Functions

Templr supports advanced templating features inspired by Helm and the [Sprig](https://masterminds.github.io/sprig/) function library, enabling powerful map manipulation, logic, and composition.

### Map Declaration and Merging

You can declare maps using `dict`, provide defaults with `default`, and merge multiple maps using `merge` or `mustMerge` (from Sprig):

```gotmpl
{{- $globalEnv := default dict .Values.images.env }}
{{- $serviceEnv := default dict .Values.mariadb.env }}
{{- $env := mustMerge $globalEnv $serviceEnv }}
```

In this example:
- `default dict .Values.images.env` ensures a map is always present, even if the value is missing.
- `mustMerge` merges two (or more) maps, with later keys taking precedence.

### Logical Helpers and Map Inspection

Sprig functions like `or`, `and`, `not`, `hasKey`, and `get` allow for expressive conditional logic and safe map access:

```gotmpl
{{- if or (not (hasKey $env "DB_HOST")) (eq (get $env "DB_HOST") "") }}
  - name: DB_HOST
    value: {{ include "drupal.fullname" . }}-mariadb
{{- end }}
```

- `hasKey` checks if a map contains a given key.
- `get` retrieves a value by key from a map.
- `or`, `not`, and `eq` are logical helpers for composing conditions.
- `include` renders another defined template with the current context.

### Notes

- `mustMerge`, `hasKey`, and `get` are provided by Sprig and are available in templr.
- Use `default (dict)` to avoid nil map errors when working with potentially missing values.
- The `include` function can be used to render sub-templates or partials you have defined elsewhere in your templates.

These capabilities make it easy to build robust, dynamic templates for complex configuration scenarios.

## 7. Additional Template Functions

Templr extends the Sprig function library with additional specialized functions for common use cases.

### Humanization Functions

Format numbers, bytes, and dates in human-readable formats:

```gotmpl
# File sizes
Disk usage: {{ 1234567890 | humanizeBytes }}
# Output: Disk usage: 1.1 GB

# Numbers with commas
Downloads: {{ 1234567 | humanizeNumber }}
# Output: Downloads: 1,234,567

# Ordinal numbers
You finished {{ 1 | ordinal }} place!
# Output: You finished 1st place!

# Relative time (requires RFC3339 timestamp)
Last updated: {{ "2024-01-01T00:00:00Z" | humanizeTime }}
# Output: Last updated: 10 months ago
```

### TOML Support

Parse and generate TOML configuration files:

```gotmpl
# Parse TOML from string or file
{{- $toml := `
name = "myapp"
version = "1.0.0"

[database]
host = "localhost"
port = 5432
` }}
{{- $config := fromToml $toml }}
App: {{ $config.name }} v{{ $config.version }}
DB: {{ $config.database.host }}:{{ $config.database.port }}

# Generate TOML
{{- $data := dict "name" "myapp" "port" 8080 }}
{{ $data | toToml }}
# Output:
# name = 'myapp'
# port = 8080
```

### Path Functions

Work with file paths and extensions:

```gotmpl
# Extract file extension
{{ pathExt "document.pdf" }}
# Output: .pdf

# Get filename without extension
{{ pathStem "archive.tar.gz" }}
# Output: archive.tar

# Normalize paths
{{ pathNormalize "foo/./bar/../baz" }}
# Output: foo/baz

# Detect MIME type from extension
{{ mimeType "image.png" }}
# Output: image/png
```

### Validation Functions

Validate common data formats:

```gotmpl
{{- if not (isEmail .contactEmail) }}
ERROR: Invalid email address
{{- end }}

{{- if not (isURL .website) }}
ERROR: Invalid URL
{{- end }}

{{- if isIPv4 .serverIP }}
Server is IPv4: {{ .serverIP }}
{{- else if isIPv6 .serverIP }}
Server is IPv6: {{ .serverIP }}
{{- end }}

{{- if not (isUUID .requestId) }}
ERROR: Invalid request ID format
{{- end }}
```

#### Real-World Example

```yaml
# values.yaml
contactEmail: admin@example.com
website: https://example.com
serverIP: 10.0.0.1
requestId: 550e8400-e29b-41d4-a716-446655440000
```

```gotmpl
# template.tpl
# Validation check
{{- if not (isEmail .contactEmail) }}
  {{- fail "Invalid contact email" }}
{{- end }}
{{- if not (isURL .website) }}
  {{- fail "Invalid website URL" }}
{{- end }}
{{- if not (isIPv4 .serverIP) }}
  {{- fail "Server IP must be IPv4" }}
{{- end }}
{{- if not (isUUID .requestId) }}
  {{- fail "Request ID must be valid UUID" }}
{{- end }}

# Configuration validated successfully
contact_email: {{ .contactEmail }}
website: {{ .website }}
server_ip: {{ .serverIP }}
request_id: {{ .requestID }}
```

### Complete Function Reference

| Function | Description | Example |
|----------|-------------|---------|
| `humanizeBytes` | Format bytes as human-readable size | `{{ 1048576 \| humanizeBytes }}` → "1.0 MB" |
| `humanizeNumber` | Add thousand separators | `{{ 1234567 \| humanizeNumber }}` → "1,234,567" |
| `humanizeTime` | Relative time format | `{{ "2024-01-01T00:00:00Z" \| humanizeTime }}` → "10 months ago" |
| `ordinal` | Convert number to ordinal | `{{ 21 \| ordinal }}` → "21st" |
| `toToml` | Serialize to TOML | `{{ $data \| toToml }}` |
| `fromToml` | Parse TOML string | `{{ $tomlStr \| fromToml }}` |
| `pathExt` | Get file extension | `{{ pathExt "file.txt" }}` → ".txt" |
| `pathStem` | Get filename without extension | `{{ pathStem "doc.pdf" }}` → "doc" |
| `pathNormalize` | Normalize path separators | `{{ pathNormalize "a/b/../c" }}` → "a/c" |
| `mimeType` | Detect MIME type from extension | `{{ mimeType "data.json" }}` → "application/json" |
| `isEmail` | Validate email address | `{{ isEmail "user@example.com" }}` → true |
| `isURL` | Validate URL | `{{ isURL "https://example.com" }}` → true |
| `isIPv4` | Check if valid IPv4 | `{{ isIPv4 "192.168.1.1" }}` → true |
| `isIPv6` | Check if valid IPv6 | `{{ isIPv6 "2001:db8::1" }}` → true |
| `isUUID` | Check if valid UUID | `{{ isUUID "550e8400-e29b-41d4-a716-446655440000" }}` → true |

---

## 8. Helper Templates and Pre-Render Variables

Templr supports the use of helper templates, typically loaded from a file named `_helpers.tpl` or from files specified with the `--helpers` flag. These helper templates can define a special template named `templr.vars` that is executed before rendering the main templates. The output of `templr.vars` should be valid YAML or JSON and is deep-merged into the root values, allowing you to transform or inject variables dynamically.

This feature enables computed variables, reusable logic, and complex preprocessing steps to be performed as part of the templating process.

### Usage Example

```gotmpl
{{- define "templr.vars" -}}
{{- $globalEnv := default (dict) .images.env -}}
{{- $serviceEnv := default (dict) .mariadb.env -}}
{{- $env := mustMerge $globalEnv $serviceEnv -}}
{{ toYaml (dict
  "env" $env
  "nameSlug" (replace (lower .name) " " "-")
) }}
{{- end -}}
```

In this example, the `templr.vars` template combines environment variables from different sources and creates a slugified version of a name. The resulting YAML output is merged into the root values before rendering the main templates.

### Additional Notes

- The `--helpers` flag controls which helper file(s) are loaded. By default, templr looks for files matching `_helpers*.tpl`.
- In single-file mode, helpers matching the glob specified by `--helpers` are loaded from the same directory as the input file.
- This mechanism enhances templr's flexibility by enabling advanced variable preparation and logic reuse.

---

## 9. Guards and Safe Access

To avoid runtime errors when accessing potentially missing data, use guards:

```gotmpl
{{ if .User }}
  User email: {{ .User.Email }}
{{ else }}
  No user data available.
{{ end }}
```

Or use the `default` function:

```gotmpl
Email: {{ default .User.Email "no-email@example.com" }}
```

---

## 10. Comments

Add comments in your template that will not appear in the output:

```gotmpl
{{/* This is a comment */}}
```

---

## 11. Putting It All Together

Example template:

```gotmpl
Hello, {{ default .Name "Guest" }}!

{{ if .Files.Glob("*.md") }}
Here are your markdown files:
{{ range .Files.Glob("*.md") }}
- {{ .Name }} ({{ len .Content }} bytes)
{{ end }}
{{ else }}
No markdown files found.
{{ end }}
```

---

## 12. Configuration Files and Project Setup

### Overview

While templr can be used with command-line flags, using configuration files (`.templr.yaml`) makes your projects more maintainable, especially for teams and CI/CD pipelines.

### Configuration File Locations

templr automatically looks for configuration in three places (in order of precedence):

1. **Specified config** via `--config` flag (highest priority)
2. **Project config**: `.templr.yaml` in the current directory
3. **User config**: `~/.config/templr/config.yaml`
4. **Built-in defaults** (lowest priority)

CLI flags always override configuration file settings.

### Basic Configuration Example

Create a `.templr.yaml` file in your project root:

```yaml
# File handling
files:
  extensions:
    - tpl
    - md      # Also treat .md files as templates
  default_values_file: ./values.yaml
  default_templates_dir: ./templates

# Template engine settings
template:
  left_delimiter: "{{"
  right_delimiter: "}}"
  default_missing: "<no value>"

# Linting rules
lint:
  fail_on_warn: true
  fail_on_undefined: true
  required_vars:
    - name
    - version

# Rendering behavior
render:
  inject_guard: true
  guard_string: "#templr generated"
  prune_empty_dirs: true
```

### How Configuration Affects Template Rendering

#### Custom Delimiters

If you need different delimiters (e.g., to avoid conflicts with other template systems):

```yaml
template:
  left_delimiter: "[["
  right_delimiter: "]]"
```

Now your templates use `[[ ]]` instead of `{{ }}`:

```gotmpl
Hello, [[ .Name ]]!
```

#### Default Missing Values

Control what appears when a variable is undefined:

```yaml
template:
  default_missing: "N/A"
```

Template:
```gotmpl
Email: {{ .User.Email }}
```

If `.User.Email` is not defined, it renders as `Email: N/A` instead of `Email: <no value>`.

#### File Extensions

Process multiple file types as templates:

```yaml
files:
  extensions:
    - tpl
    - md
    - yaml
```

Now templr will process `.tpl`, `.md`, and `.yaml` files as templates when using walk mode.

### Project Structure Best Practices

#### Recommended Layout

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

#### Helper Templates Organization

Use underscore prefix for helper templates that shouldn't be rendered directly:

```yaml
# .templr.yaml
lint:
  exclude:
    - "_*.tpl"                # Don't lint helper templates
    - "**/test/**"            # Don't lint test fixtures
```

**_helpers.tpl:**
```gotmpl
{{- define "app.fullname" -}}
{{ .Release.Name }}-{{ .Chart.Name }}
{{- end -}}

{{- define "app.labels" -}}
app: {{ .Release.Name }}
version: {{ .Chart.Version }}
{{- end -}}
```

**deployment.yaml.tpl:**
```gotmpl
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  replicas: {{ .replicas }}
```

### Environment-Specific Configurations

#### Development Setup

**.templr.dev.yaml:**
```yaml
files:
  default_values_file: ./values.dev.yaml

lint:
  fail_on_undefined: false    # Allow undefined vars in dev
  fail_on_warn: false

output:
  verbose: true               # More output for debugging
```

Usage:
```bash
templr walk --config .templr.dev.yaml --src templates --dst output
```

#### Production Setup

**.templr.prod.yaml:**
```yaml
files:
  default_values_file: ./values.prod.yaml

lint:
  fail_on_undefined: true     # Strict validation
  fail_on_warn: true
  strict_mode: true
  required_vars:              # Ensure critical vars exist
    - environment
    - version
    - database_url

render:
  inject_guard: true          # Protect generated files
```

Usage:
```bash
# Always lint before rendering in production
templr lint --config .templr.prod.yaml --src templates
templr walk --config .templr.prod.yaml --src templates --dst output
```

### Team Workflow Configuration

#### Project Standards (committed to git)

**.templr.yaml:**
```yaml
# Enforced for all team members
lint:
  required_vars:
    - projectName
    - environment
  disallow_functions:
    - env                     # No environment variable access
  fail_on_undefined: true

files:
  extensions: [yaml, tpl]
```

#### Personal Preferences (not committed)

**~/.config/templr/config.yaml:**
```yaml
# Personal developer preferences
output:
  color: always
  verbose: true

render:
  dry_run: true               # Preview changes by default
```

The team gets consistent validation and security, while individuals can customize their local experience.

### Linting and Validation

Configuration files enable powerful validation:

```yaml
lint:
  # Catch issues early
  fail_on_warn: true
  fail_on_undefined: true
  strict_mode: true

  # Security: Block dangerous functions
  disallow_functions:
    - env
    - getHostByName

  # Ensure required data exists
  required_vars:
    - appName
    - version
    - environment

  # Skip helper and test files
  exclude:
    - "_*.tpl"
    - "**/test/**"
```

Run in CI/CD:
```bash
# Fail the build if templates have issues
templr lint --src ./templates -d ./values.yaml
```

### Migration from CLI Flags

**Before** (long command):
```bash
templr walk \
  --src ./templates \
  --dst ./output \
  -d ./values.yaml \
  --strict \
  --ext md \
  --ext yaml \
  --guard "#generated" \
  --inject-guard
```

**After** (with `.templr.yaml`):
```yaml
files:
  extensions: [tpl, md, yaml]
  default_values_file: ./values.yaml
  default_templates_dir: ./templates
  default_output_dir: ./output

render:
  strict: true
  inject_guard: true
  guard_string: "#generated"
```

**New command** (simple and clear):
```bash
templr walk --src templates --dst output
```

---

## 13. Summary

- Use `{{ .Variable }}` to access data.
- Control flow with `if`, `else`, and `range`.
- Access input files with `.Files`.
- Use helpers for string and data manipulation.
- Define variables with `{{ $var := ... }}`.
- Guard against missing data with `if` or `default`.
- **Use `.templr.yaml` configuration files for consistent project settings**.
- **Organize templates with helper files and clear directory structure**.
- **Validate templates with lint mode before rendering**.

For more examples and advanced usage, explore the templr repository and CLI documentation.

Happy templating!
