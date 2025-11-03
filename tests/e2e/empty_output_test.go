package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmptyOutputSkipping(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name         string
		template     string
		data         string
		shouldSkip   bool
		expectStderr string
	}{
		{
			name:         "only ASCII spaces",
			template:     "   ",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "only newlines",
			template:     "\n\n\n",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "only tabs",
			template:     "\t\t\t",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "mixed whitespace",
			template:     " \t\n \r\n  ",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:       "template with empty variable (using --default-missing)",
			template:   "{{ .missing }}",
			data:       "",
			shouldSkip: false, // Without --default-missing "", this produces "<no value>"
		},
		{
			name:         "template producing only whitespace from conditionals",
			template:     "{{ if .false }}\ncontent\n{{ end }}",
			data:         "false: false",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "UTF-8 BOM only",
			template:     "\xEF\xBB\xBF",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "BOM with whitespace",
			template:     "\xEF\xBB\xBF   \n  ",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:         "non-breaking spaces",
			template:     "\u00A0\u00A0",
			shouldSkip:   true,
			expectStderr: "skipping empty render",
		},
		{
			name:       "single visible character",
			template:   "a",
			shouldSkip: false,
		},
		{
			name:       "visible character with whitespace",
			template:   "  a  ",
			shouldSkip: false,
		},
		{
			name:       "actual content from template",
			template:   "name: {{ .name }}",
			data:       "name: test",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := t.TempDir()
			in := filepath.Join(td, "in.tpl")
			out := filepath.Join(td, "out.txt")

			if err := os.WriteFile(in, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}

			args := []string{"-in", in, "-out", out}
			if tt.data != "" {
				dataFile := filepath.Join(td, "data.yaml")
				if err := os.WriteFile(dataFile, []byte(tt.data), 0o644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "-data", dataFile)
			}

			_, stderr, _ := run(t, bin, args...)

			// Check if file was created
			_, err := os.Stat(out)
			fileExists := err == nil

			if tt.shouldSkip {
				if fileExists {
					t.Errorf("expected output file to be skipped (not created), but it exists")
				}
				if !strings.Contains(stderr, tt.expectStderr) {
					t.Errorf("expected stderr to contain %q, got: %s", tt.expectStderr, stderr)
				}
			} else {
				if !fileExists {
					t.Errorf("expected output file to be created, but it doesn't exist")
				}
			}
		})
	}
}

func TestEmptyOutputInWalkMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create templates: one empty, one with content
	emptyTpl := filepath.Join(src, "empty.txt.tpl")
	if err := os.WriteFile(emptyTpl, []byte("   \n\n  "), 0o644); err != nil {
		t.Fatal(err)
	}

	contentTpl := filepath.Join(src, "content.txt.tpl")
	if err := os.WriteFile(contentTpl, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := run(t, bin, "-walk", "-src", src, "-dst", dst)
	if err != nil {
		t.Fatalf("templr walk failed: %v", err)
	}

	// Check that only content.txt was created
	emptyOut := filepath.Join(dst, "empty.txt")
	if _, err := os.Stat(emptyOut); err == nil {
		t.Errorf("expected empty.txt to be skipped, but it was created")
	}

	contentOut := filepath.Join(dst, "content.txt")
	if _, err := os.Stat(contentOut); err != nil {
		t.Errorf("expected content.txt to be created, but it wasn't: %v", err)
	}

	// Verify stdout mentions the content file but not the empty one
	if !strings.Contains(stdout, "content.txt") {
		t.Errorf("expected stdout to mention content.txt, got: %s", stdout)
	}
}

func TestEmptyOutputWithDryRun(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	out := filepath.Join(td, "out.txt")

	// Template that produces only whitespace
	if err := os.WriteFile(in, []byte("   \n  "), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, _ := run(t, bin, "-in", in, "-out", out, "-dry-run")

	// Should mention skipping empty
	if !strings.Contains(stdout, "[dry-run] skip empty") {
		t.Errorf("expected dry-run output to mention skipping empty render, got: %s", stdout)
	}

	// File should not exist
	if _, err := os.Stat(out); err == nil {
		t.Errorf("expected output file not to exist in dry-run mode, but it does")
	}
}

func TestEmptyOutputWithGuardInjection(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	out := filepath.Join(td, "out.txt")

	// Template that produces only whitespace
	if err := os.WriteFile(in, []byte("\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run with guard injection enabled (default)
	_, _, _ = run(t, bin, "-in", in, "-out", out)

	// File should still be skipped even though guard would be injected
	// The isEmpty check happens BEFORE guard injection
	if _, err := os.Stat(out); err == nil {
		t.Errorf("expected output file to be skipped (not created), even with guard injection")
	}
}

func TestRealWorldEmptyTemplates(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name       string
		template   string
		data       string
		shouldSkip bool
	}{
		{
			name: "conditional that evaluates to nothing",
			template: `{{ if .enabled }}
enabled: true
{{ end }}`,
			data:       "enabled: false",
			shouldSkip: true,
		},
		{
			name: "range over empty list",
			template: `{{ range .items }}
- {{ . }}
{{ end }}`,
			data:       "items: []",
			shouldSkip: true,
		},
		{
			name:       "only template comments",
			template:   `{{/* comment 1 */}}{{/* comment 2 */}}`,
			data:       "",
			shouldSkip: true, // Template produces only empty output
		},
		{
			name: "template with only comments (Go template doesn't support comments, so this is empty)",
			template: `{{/* This is a comment */}}
{{/* Another comment */}}`,
			data:       "",
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := t.TempDir()
			in := filepath.Join(td, "in.tpl")
			out := filepath.Join(td, "out.txt")

			if err := os.WriteFile(in, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}

			args := []string{"-in", in, "-out", out, "--default-missing", ""}
			if tt.data != "" {
				dataFile := filepath.Join(td, "data.yaml")
				if err := os.WriteFile(dataFile, []byte(tt.data), 0o644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "-data", dataFile)
			}

			_, _, _ = run(t, bin, args...)

			// Check if file was created
			_, err := os.Stat(out)
			fileExists := err == nil

			if tt.shouldSkip && fileExists {
				content, _ := os.ReadFile(out)
				t.Errorf("expected output to be skipped, but file was created with content: %q", content)
			} else if !tt.shouldSkip && !fileExists {
				t.Errorf("expected output file to be created, but it wasn't")
			}
		})
	}
}
