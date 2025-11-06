# Schema Validation

Schema validation in templr helps you catch configuration errors early by validating your values files against a JSON Schema before rendering templates.

## Table of Contents

- [Why Use Schema Validation](#why-use-schema-validation)
- [Quick Start](#quick-start)
- [Schema Commands](#schema-commands)
- [Configuration](#configuration)
- [Writing Schemas](#writing-schemas)
- [Common Patterns](#common-patterns)
- [CI/CD Integration](#cicd-integration)
- [Examples](#examples)

## Why Use Schema Validation

Templates can fail in two ways:

1. **Render time** - Missing keys cause strict-mode errors when the template tries to access them
2. **Runtime downstream** - Wrong value types or shapes (e.g., `replicas: "two"` instead of `replicas: 2`)

Schema validation catches these issues **before** rendering by validating that your values:

- Match expected types (string, number, boolean, object, array)
- Include all required fields
- Respect enumerations (e.g., only `dev`, `stage`, or `prod` for environment)
- Don't include unexpected/forbidden fields
- Meet constraints (minimums, maximums, patterns)

## Quick Start

### 1. Generate a Schema

Start by generating a schema from your existing values:

```bash
templr schema generate --data values.yaml -o .templr.schema.yml
```

This creates a YAML schema that describes your values structure.

### 2. Validate Your Values

```bash
templr schema validate --data values.yaml --schema .templr.schema.yml
```

Or use auto-discovery:

```bash
# Validates using .templr.schema.yml if it exists
templr schema validate --data values.yaml
```

### 3. Configure Automatic Validation

Add to `.templr.yaml`:

```yaml
schema:
  path: .templr.schema.yml
  mode: warn  # warn|error|strict
```

Now all render commands will automatically validate:

```bash
# Automatically validates before rendering
templr render -in template.tpl --data values.yaml
```

## Schema Commands

### `schema validate`

Validates data files against a schema.

```bash
templr schema validate [flags]
```

**Flags:**
- `--schema PATH` - Path to schema file (default: auto-discover)
- `--schema-mode MODE` - Validation mode: `warn`, `error`, or `strict` (default: from config or `warn`)
- `--data PATH` - Data file to validate
- `-f PATH` - Additional data files to merge
- `--set key=value` - Override values

**Examples:**

```bash
# Auto-discover schema
templr schema validate

# Explicit schema file
templr schema validate --schema my-schema.yml

# Validate merged data
templr schema validate --data base.yaml -f env/prod.yaml

# Fail on errors (not warnings)
templr schema validate --schema-mode error
```

**Modes:**

- **warn** (default) - Print validation errors as warnings, exit 0, continue rendering
- **error** - Print validation errors, exit code 8, stop rendering
- **strict** - Like error, but also fail on unknown properties (requires `additionalProperties: false` in schema)

### `schema generate`

Generates a schema by analyzing your data files.

```bash
templr schema generate [flags]
```

**Flags:**
- `-o, --output PATH` - Output schema file (default: stdout)
- `--required MODE` - Mark fields as required: `all`, `none`, or `auto` (default: from config or `auto`)
- `--additional-props BOOL` - Allow additional properties (default: true)
- `--data PATH` - Data file to analyze
- `-f PATH` - Additional data files to merge

**Examples:**

```bash
# Generate to stdout
templr schema generate --data values.yaml

# Generate to file
templr schema generate --data values.yaml -o schema.yml

# Mark all fields as required
templr schema generate --data values.yaml --required all -o schema.yml

# Disallow extra properties
templr schema generate --data values.yaml --additional-props=false -o schema.yml
```

**Required Modes:**

- **auto** (default) - Mark fields required if they have non-empty values
- **all** - Mark all fields as required
- **none** - Don't mark any fields as required

## Configuration

### `.templr.yaml`

```yaml
schema:
  path: .templr.schema.yml  # Path to schema file
  mode: warn                 # warn|error|strict

  # Schema generation defaults
  generate:
    required: auto           # all|none|auto
    additional_props: true   # Allow extra properties
    infer_types: true        # Infer types from values
```

### Auto-Discovery

When no `--schema` flag is provided, templr searches for schemas in this order:

1. `schema.path` from `.templr.yaml` config
2. `.templr.schema.yml` in current directory
3. `.templr/schema.yml` in current directory

## Writing Schemas

### Basic Structure

```yaml
$schema: http://json-schema.org/draft-07/schema#
type: object
required:
  - service
  - environment
properties:
  service:
    type: object
    # ... nested properties
  environment:
    type: string
```

### Type System

**Primitive Types:**

```yaml
name:
  type: string
  description: Service name

replicas:
  type: number
  minimum: 1
  maximum: 10

enabled:
  type: boolean
```

**Objects:**

```yaml
database:
  type: object
  required:
    - host
  properties:
    host:
      type: string
    port:
      type: number
      default: 5432
```

**Arrays:**

```yaml
servers:
  type: array
  minItems: 1
  items:
    type: object
    properties:
      hostname:
        type: string
      port:
        type: number
```

### Constraints

**Enumerations:**

```yaml
environment:
  type: string
  enum:
    - development
    - staging
    - production
```

**Number Constraints:**

```yaml
replicas:
  type: number
  minimum: 1
  maximum: 100

timeout:
  type: number
  exclusiveMinimum: 0  # Must be > 0
```

**String Constraints:**

```yaml
email:
  type: string
  pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

name:
  type: string
  minLength: 3
  maxLength: 50
```

**Array Constraints:**

```yaml
tags:
  type: array
  minItems: 1
  maxItems: 10
  uniqueItems: true
```

### Additional Properties

Control whether extra fields are allowed:

```yaml
# Allow any extra properties (default)
additionalProperties: true

# Disallow extra properties (strict mode)
additionalProperties: false

# Allow extra properties of specific type
additionalProperties:
  type: string
```

## Common Patterns

### Service Configuration

```yaml
$schema: http://json-schema.org/draft-07/schema#
type: object
required:
  - service
properties:
  service:
    type: object
    required:
      - name
      - replicas
    properties:
      name:
        type: string
        pattern: "^[a-z0-9-]+$"
        description: Service name (lowercase, alphanumeric, hyphens)

      replicas:
        type: number
        minimum: 1
        maximum: 10
        description: Number of replicas

      resources:
        type: object
        properties:
          cpu:
            type: string
            pattern: "^[0-9]+(m|)$"
            description: CPU limit (e.g., 100m, 1)

          memory:
            type: string
            pattern: "^[0-9]+(Mi|Gi)$"
            description: Memory limit (e.g., 256Mi, 1Gi)
```

### Environment-Specific Values

```yaml
$schema: http://json-schema.org/draft-07/schema#
type: object
required:
  - environment
properties:
  environment:
    type: string
    enum: [dev, stage, prod]

  config:
    type: object
    if:
      properties:
        environment:
          const: prod
    then:
      required:
        - database
        - monitoring
    else:
      required:
        - database
```

### Nested Objects with Defaults

```yaml
database:
  type: object
  required:
    - host
  properties:
    host:
      type: string

    port:
      type: number
      default: 5432

    ssl:
      type: boolean
      default: true

    pool:
      type: object
      properties:
        min:
          type: number
          default: 2
        max:
          type: number
          default: 10
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Validate Configuration

on: [pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install templr
        run: |
          curl -fsSL https://raw.githubusercontent.com/kanopi/templr/main/install.sh | sh
          sudo mv templr /usr/local/bin/

      - name: Validate schemas
        run: |
          templr schema validate \\
            --data values/base.yaml \\
            -f values/${{ matrix.env }}.yaml \\
            --schema schema.yml \\
            --schema-mode error
    strategy:
      matrix:
        env: [dev, stage, prod]
```

### GitLab CI

```yaml
validate-schemas:
  stage: validate
  image: alpine:latest
  before_script:
    - wget -qO- https://raw.githubusercontent.com/kanopi/templr/main/install.sh | sh
  script:
    - |
      for env in dev stage prod; do
        echo "Validating $env environment..."
        ./templr schema validate \\
          --data values/base.yaml \\
          -f values/$env.yaml \\
          --schema schema.yml \\
          --schema-mode error
      done
```

### Pre-commit Hook

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
if [ -f .templr.schema.yml ]; then
  echo "Validating values against schema..."
  templr schema validate --data values.yaml --schema-mode error
  if [ $? -ne 0 ]; then
    echo "Schema validation failed. Commit aborted."
    exit 1
  fi
fi
```

## Examples

### Example 1: Basic Validation

**values.yaml:**
```yaml
service:
  name: myapp
  replicas: 3
environment: production
```

**schema.yml:**
```yaml
$schema: http://json-schema.org/draft-07/schema#
type: object
required: [service, environment]
properties:
  service:
    type: object
    required: [name, replicas]
    properties:
      name:
        type: string
      replicas:
        type: number
        minimum: 1
  environment:
    type: string
    enum: [dev, stage, production]
```

**Validation:**
```bash
$ templr schema validate --data values.yaml --schema schema.yml
✓ Validation passed
```

### Example 2: Type Mismatch

**bad-values.yaml:**
```yaml
service:
  name: myapp
  replicas: "three"  # Wrong type!
environment: production
```

**Validation:**
```bash
$ templr schema validate --data bad-values.yaml --schema schema.yml
[templr:warn:schema] .service.replicas: at '/service/replicas': got string, want number
  Suggestion: set as number without quotes, e.g. `replicas: 2`
✓ Validation complete (1 warning)
```

### Example 3: Missing Required Field

**incomplete-values.yaml:**
```yaml
service:
  name: myapp
  # Missing: replicas
environment: production
```

**Validation:**
```bash
$ templr schema validate --data incomplete-values.yaml --schema schema.yml --schema-mode error
[templr:error:schema] (root): at '': missing property 'replicas'
  Suggestion: add this required field to your values file
[templr:error] validation failed
```

Exit code: 8

### Example 4: Generate and Validate Workflow

```bash
# 1. Generate initial schema from your values
$ templr schema generate --data values.yaml -o schema.yml
Generated schema -> schema.yml

# 2. Edit schema.yml to add constraints, enums, descriptions

# 3. Configure automatic validation
$ cat > .templr.yaml << EOF
schema:
  path: schema.yml
  mode: warn
EOF

# 4. Now all renders validate automatically
$ templr render -in template.tpl --data values.yaml -o output.txt
✓ Validation passed
rendered template.tpl -> output.txt
```

## Error Messages

templr provides helpful error messages with suggestions:

**Type Mismatch:**
```
[templr:error:schema] .service.replicas: expected integer, got string ("two")
  Suggestion: set as number, e.g. `replicas: 2`
```

**Missing Required:**
```
[templr:error:schema] .database.credentials.password: required property missing
  Suggestion: add this required field to your values file
```

**Enum Violation:**
```
[templr:warn:schema] .environment: must be one of [dev, stage, prod], got "development"
  Suggestion: use one of the allowed values: dev, stage, prod
```

**Unknown Property (strict mode):**
```
[templr:error:schema] .service.unknownField: property not allowed by schema
  Suggestion: remove this field or update schema to allow additional properties
```

## Exit Codes

- `0` - Validation passed or warnings only (warn mode)
- `3` - Data loading error
- `8` - Schema validation failed (error or strict mode)

## Best Practices

1. **Start with Generation** - Generate an initial schema from existing values, then refine it
2. **Add Descriptions** - Document what each field does in the schema
3. **Use Enums** - Constrain values to known-good options
4. **Set Ranges** - Use min/max for numbers, minLength/maxLength for strings
5. **Version Your Schema** - Commit `.templr.schema.yml` to git alongside your values
6. **Validate in CI** - Catch schema violations before they reach production
7. **Use Warn Mode for Development** - Get feedback without blocking, switch to error for CI
8. **Test Edge Cases** - Validate against multiple environments (dev, stage, prod)

## Troubleshooting

**Q: Schema validation passes but templates still fail?**

A: Schema validation checks the *input* data structure, not whether templates can render it. Use `templr lint` to check templates.

**Q: How do I allow optional fields?**

A: Don't include them in the `required` array. They'll be validated if present but won't cause errors if missing.

**Q: Can I use multiple schema files?**

A: Currently, templr supports one schema file per validation. You can use JSON Schema's `$ref` to reference other files within your schema.

**Q: Does schema validation work with `--set` overrides?**

A: Yes! Validation runs after all merging (defaults → data → -f → --set), so your final values are validated.

**Q: Can I validate without rendering?**

A: Yes! Use `templr schema validate` as a standalone command to check values without rendering templates.

## See Also

- [Configuration](configuration.md) - Full `.templr.yaml` reference
- [CLI Reference](cli-reference.md) - Complete command documentation
- [Examples](examples.md) - More real-world examples
