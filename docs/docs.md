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

- `.Files.Glob("pattern")` returns a list of files matching the glob pattern.
- `.Files.Get("filename")` returns the content of a specific file.

Example:

```gotmpl
{{ range .Files.Glob("*.txt") }}
- File: {{ .Name }}
{{ end }}
```

You can also access file content:

```gotmpl
{{ $content := .Files.Get "README.md" }}
Content of README:
{{ $content }}
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

## 7. Helper Templates and Pre-Render Variables

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

## 8. Guards and Safe Access

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

## 9. Comments

Add comments in your template that will not appear in the output:

```gotmpl
{{/* This is a comment */}}
```

---

## 10. Putting It All Together

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

## Summary

- Use `{{ .Variable }}` to access data.
- Control flow with `if`, `else`, and `range`.
- Access input files with `.Files`.
- Use helpers for string and data manipulation.
- Define variables with `{{ $var := ... }}`.
- Guard against missing data with `if` or `default`.

For more examples and advanced usage, explore the templr repository and CLI documentation.

Happy templating!
