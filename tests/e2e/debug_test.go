package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugFlag(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	valuesContent := `
name: Alice
count: 42
nested:
  key: value
`
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tplFile := filepath.Join(tmpDir, "test.tpl")
	template := `name: {{ .name }}
count: {{ .count }}
nested.key: {{ .nested.key }}`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "out.txt")

	// Run with --debug flag
	_, stderr, _ := run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--debug", "--inject-guard=false")

	// Verify debug output contains expected sections
	if !strings.Contains(stderr, "[DEBUG] Template Rendering Flow") {
		t.Errorf("Expected debug output to contain template rendering flow section, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Value Loading Sequence") {
		t.Errorf("Expected debug output to contain value loading sequence, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Final Merged Values") {
		t.Errorf("Expected debug output to contain final merged values, got: %s", stderr)
	}

	if !strings.Contains(stderr, "name: Alice") {
		t.Errorf("Expected debug output to show variable values, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Loading data from --data=") {
		t.Errorf("Expected debug output to show data loading, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Rendering template") {
		t.Errorf("Expected debug output to show rendering step, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Render complete") {
		t.Errorf("Expected debug output to show render completion, got: %s", stderr)
	}

	// Verify output file was still created correctly
	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := string(result)
	if !strings.Contains(got, "name: Alice") {
		t.Errorf("Expected output to contain 'name: Alice', got %q", got)
	}
	if !strings.Contains(got, "count: 42") {
		t.Errorf("Expected output to contain 'count: 42', got %q", got)
	}
}

func TestDebugWithSetOverrides(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	tplFile := filepath.Join(tmpDir, "test.tpl")
	template := `name: {{ .name }}
version: {{ .version }}`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "out.txt")

	// Run with --debug and --set flags
	_, stderr, _ := run(t, bin, "render", "-i", tplFile, "-o", outFile, "--set", "name=Bob", "--set", "version=1.2.3", "--debug", "--inject-guard=false")

	// Verify debug output shows --set overrides
	if !strings.Contains(stderr, "[DEBUG] Applying 2 --set override(s)") {
		t.Errorf("Expected debug output to mention --set overrides, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG]   → Setting name = Bob") {
		t.Errorf("Expected debug output to show name=Bob override, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG]   → Setting version = 1.2.3") {
		t.Errorf("Expected debug output to show version=1.2.3 override, got: %s", stderr)
	}

	// Verify final values show overrides
	if !strings.Contains(stderr, "name: Bob") {
		t.Errorf("Expected final values to show name: Bob, got: %s", stderr)
	}
}

func TestDebugWithHelperTemplates(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()

	// Create helper template
	helperFile := filepath.Join(tmpDir, "_helpers.tpl")
	helperContent := `{{- define "greeting" -}}
Hello, {{ .name }}!
{{- end -}}`
	if err := os.WriteFile(helperFile, []byte(helperContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main template
	tplFile := filepath.Join(tmpDir, "test.tpl")
	template := `{{ include "greeting" . }}`
	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values file
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesFile, []byte("name: World\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "out.txt")

	// Run with --debug
	_, stderr, _ := run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--debug", "--inject-guard=false")

	// Verify debug output shows helper loading
	if !strings.Contains(stderr, "[DEBUG] Looking for helper templates:") {
		t.Errorf("Expected debug output to mention helper templates, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Found 1 helper template(s)") {
		t.Errorf("Expected debug output to show 1 helper found, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG]   → Loading helper: _helpers.tpl") {
		t.Errorf("Expected debug output to show helper being loaded, got: %s", stderr)
	}

	// Verify output is correct
	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(result))
	if got != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %q", got)
	}
}

func TestDebugWithTemplrVars(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()

	// Create helper with templr.vars
	helperFile := filepath.Join(tmpDir, "_helpers.tpl")
	helperContent := `{{- define "templr.vars" -}}
computed: {{ .base | upper }}
doubled: {{ mul .count 2 }}
{{- end -}}`
	if err := os.WriteFile(helperFile, []byte(helperContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main template
	tplFile := filepath.Join(tmpDir, "test.tpl")
	template := `computed: {{ .computed }}
doubled: {{ .doubled }}`
	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values file
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesFile, []byte("base: hello\ncount: 21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "out.txt")

	// Run with --debug
	_, stderr, _ := run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--debug", "--inject-guard=false")

	// Verify debug output shows templr.vars processing
	if !strings.Contains(stderr, "[DEBUG] Checking for templr.vars template") {
		t.Errorf("Expected debug output to check for templr.vars, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG]   → templr.vars executed, values updated") {
		t.Errorf("Expected debug output to show templr.vars execution, got: %s", stderr)
	}

	if !strings.Contains(stderr, "[DEBUG] Values After templr.vars") {
		t.Errorf("Expected debug output to show values after templr.vars, got: %s", stderr)
	}

	// Verify values after templr.vars contain computed values
	// Extract the section after "Values After templr.vars"
	if strings.Contains(stderr, "Values After templr.vars") {
		parts := strings.Split(stderr, "Values After templr.vars")
		if len(parts) > 1 {
			afterVars := parts[1]
			if !strings.Contains(afterVars, "computed: HELLO") {
				t.Errorf("Expected computed value in post-templr.vars output, got: %s", afterVars)
			}
			if !strings.Contains(afterVars, "doubled: 42") {
				t.Errorf("Expected doubled value in post-templr.vars output, got: %s", afterVars)
			}
		}
	}

	// Verify output is correct
	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := string(result)
	if !strings.Contains(got, "computed: HELLO") {
		t.Errorf("Expected output to contain 'computed: HELLO', got %q", got)
	}
	if !strings.Contains(got, "doubled: 42") {
		t.Errorf("Expected output to contain 'doubled: 42', got %q", got)
	}
}

func TestDebugOutputToStderr(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	tplFile := filepath.Join(tmpDir, "test.tpl")
	template := `test`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "out.txt")

	// Run with --debug
	stdout, stderr, _ := run(t, bin, "render", "-i", tplFile, "-o", outFile, "--debug", "--inject-guard=false")

	// Verify debug output goes to stderr, not stdout
	if strings.Contains(stdout, "[DEBUG]") {
		t.Errorf("Debug output should go to stderr, not stdout. Got stdout: %s", stdout)
	}

	if !strings.Contains(stderr, "[DEBUG]") {
		t.Errorf("Debug output should be in stderr. Got stderr: %s", stderr)
	}

	// Verify output file doesn't contain debug info
	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(result), "[DEBUG]") {
		t.Errorf("Output file should not contain debug info, got: %s", string(result))
	}
}
