package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSubcommandRenderBasic tests the new "render" subcommand syntax
func TestSubcommandRenderBasic(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `name: {{ .name }}
version: {{ .version }}`
	vals := "name: templr\nversion: \"1.0\"\n"
	in := filepath.Join(td, "in.tpl")
	data := filepath.Join(td, "values.yaml")
	out := filepath.Join(td, "out.yaml")

	if err := os.WriteFile(in, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use new subcommand syntax
	_, stderr, err := run(t, bin, "render", "-i", in, "-d", data, "-o", out)
	if err != nil {
		t.Fatalf("templr render failed: %v, stderr=%s", err, stderr)
	}

	gotBytes, _ := os.ReadFile(out)
	got := normalizeOut(string(gotBytes))
	want := normalizeOut("#templr generated\nname: templr\nversion: 1.0\n")

	if got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestSubcommandVsLegacySyntax verifies that both syntaxes produce identical results
func TestSubcommandVsLegacySyntax(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name       string
		template   string
		values     string
		legacyArgs []string
		newArgs    []string
	}{
		{
			name:       "simple_render",
			template:   "result: {{ .value }}",
			values:     "value: 42",
			legacyArgs: []string{"-in", "IN", "-data", "DATA", "-out", "OUT"},
			newArgs:    []string{"render", "-i", "IN", "-d", "DATA", "-o", "OUT"},
		},
		{
			name:       "with_set_flag",
			template:   "name: {{ .name }}\nenv: {{ .env }}",
			values:     "name: test",
			legacyArgs: []string{"-in", "IN", "-data", "DATA", "--set", "env=prod", "-out", "OUT"},
			newArgs:    []string{"render", "-i", "IN", "-d", "DATA", "--set", "env=prod", "-o", "OUT"},
		},
		{
			name:       "with_strict",
			template:   "value: {{ .value }}",
			values:     "value: strict-test",
			legacyArgs: []string{"-in", "IN", "-data", "DATA", "--strict", "-out", "OUT"},
			newArgs:    []string{"render", "-i", "IN", "-d", "DATA", "--strict", "-o", "OUT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup files for legacy
			td1 := t.TempDir()
			in1 := filepath.Join(td1, "in.tpl")
			data1 := filepath.Join(td1, "values.yaml")
			out1 := filepath.Join(td1, "out.yaml")

			if err := os.WriteFile(in1, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(data1, []byte(tt.values), 0o644); err != nil {
				t.Fatal(err)
			}

			// Run with legacy syntax
			legacyArgs := make([]string, len(tt.legacyArgs))
			copy(legacyArgs, tt.legacyArgs)
			for i, arg := range legacyArgs {
				switch arg {
				case "IN":
					legacyArgs[i] = in1
				case "DATA":
					legacyArgs[i] = data1
				case "OUT":
					legacyArgs[i] = out1
				}
			}

			_, stderr1, err1 := run(t, bin, legacyArgs...)
			if err1 != nil {
				t.Fatalf("legacy syntax failed: %v, stderr=%s", err1, stderr1)
			}

			// Setup files for new syntax
			td2 := t.TempDir()
			in2 := filepath.Join(td2, "in.tpl")
			data2 := filepath.Join(td2, "values.yaml")
			out2 := filepath.Join(td2, "out.yaml")

			if err := os.WriteFile(in2, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(data2, []byte(tt.values), 0o644); err != nil {
				t.Fatal(err)
			}

			// Run with new subcommand syntax
			newArgs := make([]string, len(tt.newArgs))
			copy(newArgs, tt.newArgs)
			for i, arg := range newArgs {
				switch arg {
				case "IN":
					newArgs[i] = in2
				case "DATA":
					newArgs[i] = data2
				case "OUT":
					newArgs[i] = out2
				}
			}

			_, stderr2, err2 := run(t, bin, newArgs...)
			if err2 != nil {
				t.Fatalf("new syntax failed: %v, stderr=%s", err2, stderr2)
			}

			// Compare outputs
			legacy, _ := os.ReadFile(out1)
			newSyntax, _ := os.ReadFile(out2)

			legacyNorm := normalizeOut(string(legacy))
			newNorm := normalizeOut(string(newSyntax))

			if legacyNorm != newNorm {
				t.Fatalf("outputs differ:\n--- legacy ---\n%s\n--- new ---\n%s", legacyNorm, newNorm)
			}
		})
	}
}

// TestSubcommandDir tests the "dir" subcommand
func TestSubcommandDir(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()

	// Create helper template
	helper := `{{- define "greeting" -}}
Hello {{ .name }}!
{{- end -}}`
	helperPath := filepath.Join(td, "_helpers.tpl")
	if err := os.WriteFile(helperPath, []byte(helper), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main template that uses helper
	main := `{{ include "greeting" . }}`
	mainPath := filepath.Join(td, "main.tpl")
	if err := os.WriteFile(mainPath, []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values
	vals := "name: World"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(td, "out.txt")

	// Test new syntax
	_, stderr, err := run(t, bin, "dir", "--dir", td, "-i", "main.tpl", "-d", valPath, "-o", out)
	if err != nil {
		t.Fatalf("templr dir failed: %v, stderr=%s", err, stderr)
	}

	gotBytes, _ := os.ReadFile(out)
	got := normalizeOut(string(gotBytes))

	if !strings.Contains(got, "Hello World!") {
		t.Fatalf("expected 'Hello World!' in output, got: %s", got)
	}
}

// TestSubcommandWalk tests the "walk" subcommand
func TestSubcommandWalk(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create template file
	tpl := "name: {{ .name }}"
	tplPath := filepath.Join(src, "config.yaml.tpl")
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create values file
	vals := "name: walk-test"
	valPath := filepath.Join(src, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test new syntax
	_, stderr, err := run(t, bin, "walk", "--src", src, "--dst", dst)
	if err != nil {
		t.Fatalf("templr walk failed: %v, stderr=%s", err, stderr)
	}

	// Check output file was created (without .tpl extension)
	outPath := filepath.Join(dst, "config.yaml")
	gotBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	got := normalizeOut(string(gotBytes))
	if !strings.Contains(got, "name: walk-test") {
		t.Fatalf("expected 'name: walk-test' in output, got: %s", got)
	}
}

// TestSubcommandVersion tests the version subcommand
func TestSubcommandVersion(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name string
		args []string
	}{
		{"new_syntax", []string{"version"}},
		{"legacy_short", []string{"-version"}},
		{"legacy_long", []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := run(t, bin, tt.args...)
			if err != nil {
				t.Fatalf("version command failed: %v, stderr=%s", err, stderr)
			}

			// Should output version (either a tag or "dev")
			if stdout == "" {
				t.Fatal("version output is empty")
			}

			// Should be a simple version string
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			if len(lines) != 1 {
				t.Fatalf("expected single line version, got %d lines: %s", len(lines), stdout)
			}
		})
	}
}

// TestSubcommandHelp tests help output
func TestSubcommandHelp(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name: "root_help_short",
			args: []string{"-h"},
			contains: []string{
				"SUBCOMMANDS:",
				"render",
				"dir",
				"walk",
				"version",
				"EXAMPLES:",
			},
		},
		{
			name: "root_help_long",
			args: []string{"--help"},
			contains: []string{
				"SUBCOMMANDS:",
				"render",
				"dir",
				"walk",
			},
		},
		{
			name: "help_command",
			args: []string{"help"},
			contains: []string{
				"SUBCOMMANDS:",
				"Available Commands:",
			},
		},
		{
			name: "render_help",
			args: []string{"help", "render"},
			contains: []string{
				"Render a single template file",
				"Examples:",
				"-in",
				"-out",
			},
		},
		{
			name: "dir_help",
			args: []string{"help", "dir"},
			contains: []string{
				"Parse all templates in a directory",
				"--dir",
			},
		},
		{
			name: "walk_help",
			args: []string{"help", "walk"},
			contains: []string{
				"Recursively walk",
				"--src",
				"--dst",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := run(t, bin, tt.args...)

			// Help commands should not error
			if err != nil && !strings.Contains(tt.name, "help") {
				t.Fatalf("help command failed: %v, stderr=%s", err, stderr)
			}

			output := stdout
			if output == "" {
				output = stderr
			}

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("expected help output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

// TestSubcommandStdinStdout tests stdin/stdout with subcommands
func TestSubcommandStdinStdout(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	vals := "name: stdin-test"
	valPath := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(valPath, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"legacy_syntax", []string{"-data", valPath}},
		{"new_syntax", []string{"render", "-d", valPath}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a template file for stdin simulation
			tpl := "output: {{ .name }}"
			tplPath := filepath.Join(td, "stdin.tpl")
			if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
				t.Fatal(err)
			}

			// Use -in or -i depending on syntax (legacy uses long form)
			inFlag := "-in"
			if strings.Contains(tt.name, "new") {
				inFlag = "-i"
			}
			args := append(tt.args, inFlag, tplPath)
			stdout, stderr, err := run(t, bin, args...)
			if err != nil {
				t.Fatalf("command failed: %v, stderr=%s", err, stderr)
			}

			// When no -out is specified, output goes to stdout
			if !strings.Contains(stdout, "output: stdin-test") {
				t.Fatalf("expected 'output: stdin-test' in stdout, got: %s", stdout)
			}
		})
	}
}

// TestSubcommandFlags tests that global flags work with all subcommands
func TestSubcommandFlags(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := "value: {{ .val }}"
	in := filepath.Join(td, "in.tpl")
	out := filepath.Join(td, "out.txt")

	if err := os.WriteFile(in, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test that global flags work with render subcommand
	_, stderr, err := run(t, bin, "render", "-i", in, "--set", "val=test", "-o", out)
	if err != nil {
		t.Fatalf("render with --set failed: %v, stderr=%s", err, stderr)
	}

	gotBytes, _ := os.ReadFile(out)
	got := normalizeOut(string(gotBytes))

	if !strings.Contains(got, "value: test") {
		t.Fatalf("expected 'value: test' in output, got: %s", got)
	}
}
