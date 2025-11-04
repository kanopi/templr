package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigFileLoading tests basic config file loading
func TestConfigFileLoading(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config file
	config := `lint:
  fail_on_warn: true
  output_format: text`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create template with undefined variable
	tpl := `name: {{ .name }}
undefined: {{ .notdefined }}`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values
	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory so .templr.yaml is found
	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint - should fail with exit 6 because fail_on_warn is true in config
	_, _, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err == nil {
		t.Fatal("expected lint to fail due to fail_on_warn in config")
	}
}

// TestConfigExcludePatterns tests file exclusion patterns
func TestConfigExcludePatterns(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config with exclude patterns
	config := `lint:
  exclude:
    - "_*.tpl"
    - "**/*.backup.tpl"`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create excluded file with syntax error (should be skipped)
	excluded := `{{ if .broken` // Missing {{end}}
	excludedPath := filepath.Join(td, "_helpers.tpl")
	if err := os.WriteFile(excludedPath, []byte(excluded), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create normal file (should be linted)
	normal := `name: {{ .name }}`
	normalPath := filepath.Join(td, "normal.tpl")
	if err := os.WriteFile(normalPath, []byte(normal), 0o644); err != nil {
		t.Fatal(err)
	}

	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint --src - should NOT fail because _helpers.tpl is excluded
	stdout, stderr, err := run(t, bin, "lint", "--src", td, "-d", valPath)
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	output := stdout + stderr
	if strings.Contains(output, "_helpers.tpl") {
		t.Fatalf("excluded file should not be in output: %s", output)
	}
}

// TestConfigDisallowFunctions tests disallowed function checking
//
//nolint:dupl // Test setup similar to other config tests
func TestConfigDisallowFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config that disallows 'env' function
	config := `lint:
  disallow_functions:
    - env
    - exec`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create template using env function
	tpl := `name: {{ .name }}
env_var: {{ env "HOME" }}`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint - should fail because 'env' is disallowed
	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err == nil {
		t.Fatal("expected lint to fail due to disallowed function")
	}

	output := stdout + stderr
	if !strings.Contains(output, "disallowed function") {
		t.Fatalf("expected disallowed function error, got: %s", output)
	}
	if !strings.Contains(output, "env") {
		t.Fatalf("expected 'env' in error message, got: %s", output)
	}
}

// TestConfigRequiredVars tests required variable validation
//
//nolint:dupl // Test setup similar to other config tests
func TestConfigRequiredVars(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config with required variables
	config := `lint:
  required_vars:
    - name
    - version
    - environment`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create simple template
	tpl := `name: {{ .name }}`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values missing 'environment'
	vals := "name: test\nversion: \"1.0\"\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint - should fail because 'environment' is missing
	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err == nil {
		t.Fatal("expected lint to fail due to missing required variable")
	}

	output := stdout + stderr
	if !strings.Contains(output, "required variable") {
		t.Fatalf("expected required variable error, got: %s", output)
	}
	if !strings.Contains(output, "environment") {
		t.Fatalf("expected 'environment' in error message, got: %s", output)
	}
}

// TestConfigFailOnUndefined tests fail_on_undefined setting
func TestConfigFailOnUndefined(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config with fail_on_undefined
	config := `lint:
  fail_on_undefined: true`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create template with undefined variable
	tpl := `name: {{ .name }}
undefined: {{ .notdefined }}`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint - should fail with exit 7 (error) instead of 0 (warning)
	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err == nil {
		t.Fatal("expected lint to fail due to fail_on_undefined")
	}

	output := stdout + stderr
	if !strings.Contains(output, "[lint:error:undefined]") {
		t.Fatalf("expected error severity for undefined variable, got: %s", output)
	}
}

// TestConfigCustomDelimiters tests custom delimiter configuration
func TestConfigCustomDelimiters(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config with custom delimiters
	config := `template:
  left_delimiter: "[["
  right_delimiter: "]]"`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create template using custom delimiters
	tpl := `name: [[ .name ]]`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint - should succeed with custom delimiters from config
	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	if !strings.Contains(stdout+stderr, "No issues found") {
		t.Fatalf("expected no issues with custom delimiters, got: %s", stdout+stderr)
	}
}

// TestConfigCLIOverride tests that CLI flags override config
func TestConfigCLIOverride(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create config with fail_on_warn: false
	config := `lint:
  fail_on_warn: false`

	configPath := filepath.Join(td, ".templr.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create template with undefined variable
	tpl := `name: {{ .name }}
undefined: {{ .notdefined }}`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Run lint with --fail-on-warn CLI flag (should override config)
	_, _, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath, "--fail-on-warn")
	if err == nil {
		t.Fatal("expected lint to fail because CLI --fail-on-warn overrides config")
	}
}
