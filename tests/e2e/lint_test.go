package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLintValidTemplate tests linting a valid template
func TestLintValidTemplate(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
version: {{ .version }}`
	vals := "name: test\nversion: \"1.0\"\n"

	tplPath := filepath.Join(td, "valid.tpl")
	valPath := filepath.Join(td, "values.yaml")

	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	if err != nil {
		t.Fatalf("lint failed: %v, stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, "No issues found") {
		t.Fatalf("expected 'No issues found', got: %s", stdout)
	}
}

// TestLintSyntaxError tests linting a template with syntax errors
func TestLintSyntaxError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	// Missing {{end}}
	tpl := `name: {{ .name }}
{{ if .enabled }}
  status: active`

	tplPath := filepath.Join(td, "invalid.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath)
	if err == nil {
		t.Fatal("expected lint to fail on syntax error")
	}

	output := stdout + stderr
	if !strings.Contains(output, "[lint:error:parse]") {
		t.Fatalf("expected parse error, got: %s", output)
	}
	if !strings.Contains(output, "Found 1 error") {
		t.Fatalf("expected error count, got: %s", output)
	}
}

// TestLintUndefinedVariables tests undefined variable detection
func TestLintUndefinedVariables(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
missing: {{ .notdefined }}
nested: {{ .deep.value }}`
	vals := "name: test\n"

	tplPath := filepath.Join(td, "undefined.tpl")
	valPath := filepath.Join(td, "values.yaml")

	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath)
	// Should exit with 0 (warnings don't fail by default)
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	output := stdout + stderr
	if !strings.Contains(output, "[lint:warn:undefined]") {
		t.Fatalf("expected undefined warning, got: %s", output)
	}
	if !strings.Contains(output, ".notdefined") {
		t.Fatalf("expected .notdefined in output, got: %s", output)
	}
	if !strings.Contains(output, ".deep.value") {
		t.Fatalf("expected .deep.value in output, got: %s", output)
	}
}

// TestLintFailOnWarn tests --fail-on-warn flag
func TestLintFailOnWarn(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
missing: {{ .undefined }}`
	vals := "name: test\n"

	tplPath := filepath.Join(td, "warn.tpl")
	valPath := filepath.Join(td, "values.yaml")

	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath, "--fail-on-warn")
	if err == nil {
		t.Fatal("expected lint to fail with --fail-on-warn")
	}

	// Exit code should be 6 (ExitLintWarn)
	// We can't easily check exit code in this test framework,
	// but we verified it fails which is the important part
}

// TestLintDirectoryMode tests linting all templates in a directory
func TestLintDirectoryMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Valid template
	valid := "name: {{ .name }}"
	if err := os.WriteFile(filepath.Join(td, "valid.tpl"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}

	// Invalid template
	invalid := "name: {{ .name }\nmissing: {{ if .bad }}"
	if err := os.WriteFile(filepath.Join(td, "invalid.tpl"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}

	// Values
	vals := "name: test\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "--dir", td, "-d", valPath)
	if err == nil {
		t.Fatal("expected lint to fail on invalid template")
	}

	output := stdout + stderr
	if !strings.Contains(output, "[lint:error:parse]") {
		t.Fatalf("expected parse error, got: %s", output)
	}
	if !strings.Contains(output, "invalid.tpl") {
		t.Fatalf("expected invalid.tpl in output, got: %s", output)
	}
}

// TestLintWalkMode tests linting an entire directory tree
func TestLintWalkMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	src := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create nested directory
	subdir := filepath.Join(src, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Template in root
	tpl1 := "root: {{ .name }}"
	if err := os.WriteFile(filepath.Join(src, "root.tpl"), []byte(tpl1), 0o644); err != nil {
		t.Fatal(err)
	}

	// Template in subdirectory
	tpl2 := "sub: {{ .value }}"
	if err := os.WriteFile(filepath.Join(subdir, "sub.tpl"), []byte(tpl2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Values
	vals := "name: test\n"
	valPath := filepath.Join(src, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "--src", src, "-d", valPath)
	// Should exit with 0 (warnings about .value being undefined)
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	output := stdout + stderr
	if !strings.Contains(output, ".value") {
		t.Fatalf("expected .value undefined warning, got: %s", output)
	}
}

// TestLintJSONOutput tests JSON output format
func TestLintJSONOutput(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
missing: {{ .undefined }}`
	vals := "name: test\n"

	tplPath := filepath.Join(td, "test.tpl")
	valPath := filepath.Join(td, "values.yaml")

	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath, "--format", "json")
	if err != nil {
		t.Fatalf("lint failed: %v, stderr=%s", err, stderr)
	}

	// Parse JSON output
	var result struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Issues   []struct {
			Severity string `json:"severity"`
			Category string `json:"category"`
			File     string `json:"file"`
			Line     int    `json:"line"`
			Message  string `json:"message"`
		} `json:"issues"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v, output=%s", err, stdout)
	}

	if result.Warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", result.Warnings)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Severity != "warn" {
		t.Fatalf("expected warn severity, got %s", result.Issues[0].Severity)
	}
	if result.Issues[0].Category != "undefined" {
		t.Fatalf("expected undefined category, got %s", result.Issues[0].Category)
	}
}

// TestLintNoUndefCheck tests --no-undefined-check flag
func TestLintNoUndefCheck(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
missing: {{ .undefined }}`
	vals := "name: test\n"

	tplPath := filepath.Join(td, "test.tpl")
	valPath := filepath.Join(td, "values.yaml")

	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "-d", valPath, "--no-undefined-check")
	if err != nil {
		t.Fatalf("lint failed: %v, stderr=%s", err, stderr)
	}

	output := stdout + stderr
	if strings.Contains(output, "[lint:warn:undefined]") {
		t.Fatalf("expected no undefined warnings with --no-undefined-check, got: %s", output)
	}
	if !strings.Contains(output, "No issues found") {
		t.Fatalf("expected no issues, got: %s", output)
	}
}

// TestLintHelperTemplates tests linting templates with helpers/includes
func TestLintHelperTemplates(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Helper template
	helper := `{{- define "greeting" -}}
Hello {{ .name }}!
{{- end -}}`
	if err := os.WriteFile(filepath.Join(td, "_helpers.tpl"), []byte(helper), 0o644); err != nil {
		t.Fatal(err)
	}

	// Main template using helper
	main := `{{ include "greeting" . }}`
	if err := os.WriteFile(filepath.Join(td, "main.tpl"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}

	// Values
	vals := "name: World\n"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "--dir", td, "-d", valPath)
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	if !strings.Contains(stdout+stderr, "No issues found") {
		t.Fatalf("expected no issues with helper templates, got: %s", stdout+stderr)
	}
}

// TestLintGitHubActionsOutput tests GitHub Actions output format
func TestLintGitHubActionsOutput(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	// Missing {{end}}
	tpl := `{{ if .enabled }}
  active
`
	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "--format", "github-actions")
	if err == nil {
		t.Fatal("expected lint to fail")
	}

	output := stdout + stderr
	if !strings.Contains(output, "::error file=") {
		t.Fatalf("expected GitHub Actions error format, got: %s", output)
	}
	if !strings.Contains(output, tplPath) {
		t.Fatalf("expected file path in output, got: %s", output)
	}
}

// TestLintWithSetFlag tests linting with --set overrides
func TestLintWithSetFlag(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
env: {{ .environment }}`

	tplPath := filepath.Join(td, "test.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use --set to provide values instead of data file
	stdout, stderr, err := run(t, bin, "lint", "-i", tplPath, "--set", "name=test", "--set", "environment=prod")
	if err != nil {
		t.Fatalf("lint failed: %v, output=%s", err, stdout+stderr)
	}

	if !strings.Contains(stdout+stderr, "No issues found") {
		t.Fatalf("expected no issues with --set values, got: %s", stdout+stderr)
	}
}
