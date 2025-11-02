# templr

## Overview

templr is a Go-based templating CLI inspired by Helm and Go's `text/template` package. It allows you to render templates from single files or entire directories, providing powerful features to manage complex templating workflows. templr is designed to be flexible and easy to use, making it ideal for generating configuration files, manifests, or any text-based output from templates.

## Features

- **Multi-file rendering**: Render single template files or entire directories of templates.
- **Walk mode**: Recursively walk through directories and render all templates found.
- **`.Files` API**: Access files within the template directory during rendering.
- **Strict mode**: Enforce strict template parsing and execution to catch errors early.
- **Guards**: Use the `--guard` flag to conditionally skip rendering files based on template output.
- **Dry-run**: Preview rendered output without writing files to disk using the `--dry-run` flag.
- **Pruning empty directories**: Automatically detect and prune directories containing only whitespace or empty output.
- **Flexible data input**: Pass data via `--set` flags or load from JSON/YAML files with `--data`.
- **Custom extensions**: Use the `--ext` flag to include additional template file extensions (e.g., md, txt). `.tpl` is always included by default.

## Installation

### Download Latest Release

Download the latest pre-built binary for your platform from the [GitHub Releases](https://github.com/kanopi/templr/releases) page. Extract the archive and place the `templr` binary in your system PATH.

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
docker run --rm -v $(pwd):/templates kanopi/templr -d /templates
```

Or to run a single template file:

```bash
docker run --rm -v $(pwd):/templates kanopi/templr -f /templates/template.tpl
```

## Usage and Scenarios

templr supports rendering templates in various modes and includes a full suite of examples to help you get started and verify functionality.

### Rendering Modes

- **Single-file mode**: Render a single template file.

  ```bash
  templr -f path/to/template.tmpl
  ```

- **Directory mode**: Render all templates in a directory.

  ```bash
  templr -d path/to/templates/
  ```

- **Walk mode**: Recursively walk through a directory and render all templates.

  ```bash
  templr --walk -d path/to/templates/
  ```

  ```bash
  templr --walk -d path/to/templates/ --ext md --ext txt
  ```

### Custom Template Extensions

By default, templr processes files ending in `.tpl`. You can extend this behavior with the `--ext` flag to include additional text-based extensions such as `md`, `txt`, `html`, etc. This allows you to use templr for Markdown, documentation, or configuration file templating.

### Common Command-line Flags

- `-f, --file`: Specify a single template file to render.
- `-d, --dir`: Specify a directory containing templates.
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

### Versioning

templr includes a built-in `-version` flag to display the current version of the binary.

```bash
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

The `--guard` flag enables conditional rendering of templates. When enabled, templr evaluates the rendered output of a file and skips writing it if certain conditions are met (e.g., if the output is empty or does not meet criteria). This helps prevent overwriting files unnecessarily and improves rendering efficiency.

### Dry Run

Using the `--dry-run` flag renders templates and outputs the result to stdout or logs without writing any files to disk. This is useful for previewing changes or debugging templates before applying them.

### Skipping Empty Output

templr automatically detects output that contains only whitespace or is empty and prunes such files and their parent directories. This behavior helps keep your output clean by removing unnecessary empty files and directories generated by templates that produce no meaningful content.

## Documentation

For a full reference of templr’s templating syntax, variables, conditionals, functions, and `.Files` API, see the [docs.md](./docs/docs.md) file.

## License

This project is licensed under the [Your License Here]. See the LICENSE file for details.
