package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// getExitCode extracts the exit code from an exec.ExitError, or returns 0 if err is nil.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func TestExitCodes_General_WalkMissingArgs(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	// -walk requires both -src and -dst
	_, stderr, err := run(t, bin, "-walk")
	exitCode := getExitCode(err)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (ExitGeneral), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:args]") {
		t.Errorf("expected error kind 'args', stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "-walk requires -src and -dst") {
		t.Errorf("expected walk requirements message, stderr=%s", stderr)
	}
}

func TestExitCodes_General_InvalidSetFormat(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	if err := os.WriteFile(in, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// --set without '=' should fail with ExitGeneral
	_, stderr, err := run(t, bin, "-in", in, "--set", "noequals")
	exitCode := getExitCode(err)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (ExitGeneral), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:args]") {
		t.Errorf("expected error kind 'args', stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "--set expects key=value") {
		t.Errorf("expected set format message, stderr=%s", stderr)
	}
}

func TestExitCodes_DataError_InvalidDataFile(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	badData := filepath.Join(td, "bad.yaml")

	if err := os.WriteFile(in, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create invalid YAML
	if err := os.WriteFile(badData, []byte("invalid: yaml:\n  - broken\n   bad indent"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-data", badData)
	exitCode := getExitCode(err)

	if exitCode != 3 {
		t.Errorf("expected exit code 3 (ExitDataError), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:data]") {
		t.Errorf("expected error kind 'data', stderr=%s", stderr)
	}
}

func TestExitCodes_DataError_MissingDataFile(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	missingData := filepath.Join(td, "nonexistent.yaml")

	if err := os.WriteFile(in, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-data", missingData)
	exitCode := getExitCode(err)

	if exitCode != 3 {
		t.Errorf("expected exit code 3 (ExitDataError), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:data]") {
		t.Errorf("expected error kind 'data', stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "load data:") {
		t.Errorf("expected 'load data' message, stderr=%s", stderr)
	}
}

func TestExitCodes_TemplateError_ParseError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")

	// Invalid template syntax
	if err := os.WriteFile(in, []byte("{{ .value"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in)
	exitCode := getExitCode(err)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (ExitTemplateError), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:parse]") {
		t.Errorf("expected error kind 'parse', stderr=%s", stderr)
	}
}

func TestExitCodes_TemplateError_RenderError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")

	// Template that uses 'fail' helper - parses fine but fails during render
	if err := os.WriteFile(in, []byte("{{ fail \"intentional render failure\" }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in)
	exitCode := getExitCode(err)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (ExitTemplateError), got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:render]") {
		t.Errorf("expected error kind 'render', stderr=%s", stderr)
	}
}

func TestExitCodes_StrictError_MissingKey(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")

	// Template references undefined key
	if err := os.WriteFile(in, []byte("value: {{ .missingKey }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-strict")
	exitCode := getExitCode(err)

	if exitCode != 4 {
		t.Errorf("expected exit code 4 (ExitStrictError), got %d", exitCode)
	}

	// Check for enhanced error format (with or without color codes)
	if !strings.Contains(stderr, "Strict Mode Error") {
		t.Errorf("expected 'Strict Mode Error' in enhanced format, stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "missingKey") {
		t.Errorf("expected missing key name in error output, stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "Tip:") {
		t.Errorf("expected helpful tip in error output, stderr=%s", stderr)
	}
}

func TestExitCodes_StrictError_NoColor(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")

	// Template references undefined key
	if err := os.WriteFile(in, []byte("value: {{ .missingKey }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-strict", "--no-color")
	exitCode := getExitCode(err)

	if exitCode != 4 {
		t.Errorf("expected exit code 4 (ExitStrictError), got %d", exitCode)
	}

	// Check for enhanced error format without ANSI color codes
	if !strings.Contains(stderr, "Strict Mode Error") {
		t.Errorf("expected 'Strict Mode Error' in output, stderr=%s", stderr)
	}

	// Ensure no ANSI color codes are present
	if strings.Contains(stderr, "\033[") {
		t.Errorf("expected no ANSI color codes with --no-color, stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "missingKey") {
		t.Errorf("expected missing key name in error output, stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "Tip:") {
		t.Errorf("expected helpful tip in error output, stderr=%s", stderr)
	}
}

func TestExitCodes_GuardWarning(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	out := filepath.Join(td, "out.yaml")

	if err := os.WriteFile(in, []byte("content: updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing file WITHOUT guard marker
	if err := os.WriteFile(out, []byte("content: original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-out", out, "--guard", "#templr generated")

	// Should succeed (exit 0) but emit warning
	exitCode := getExitCode(err)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for guard warning, got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:warn:guard]") {
		t.Errorf("expected warning kind 'guard', stderr=%s", stderr)
	}

	if !strings.Contains(stderr, "skip (guard missing)") {
		t.Errorf("expected guard skip message, stderr=%s", stderr)
	}
}

func TestExitCodes_WalkMode_DataError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")
	badData := filepath.Join(td, "bad.yaml")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create invalid YAML
	if err := os.WriteFile(badData, []byte("invalid:\n  yaml:\n bad"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-walk", "-src", src, "-dst", dst, "-data", badData)
	exitCode := getExitCode(err)

	if exitCode != 3 {
		t.Errorf("expected exit code 3 (ExitDataError) in walk mode, got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:data]") {
		t.Errorf("expected error kind 'data' in walk mode, stderr=%s", stderr)
	}
}

func TestExitCodes_WalkMode_TemplateParseError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create template with syntax error
	tplFile := filepath.Join(src, "bad.tpl")
	if err := os.WriteFile(tplFile, []byte("{{ .value"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-walk", "-src", src, "-dst", dst)
	exitCode := getExitCode(err)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (ExitTemplateError) in walk mode, got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:parse]") {
		t.Errorf("expected error kind 'parse' in walk mode, stderr=%s", stderr)
	}
}

func TestExitCodes_DirMode_DataError(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	badData := filepath.Join(td, "bad.yaml")

	// Create invalid YAML
	if err := os.WriteFile(badData, []byte("bad:\nyaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-dir", td, "-data", badData)
	exitCode := getExitCode(err)

	if exitCode != 3 {
		t.Errorf("expected exit code 3 (ExitDataError) in dir mode, got %d", exitCode)
	}

	if !strings.Contains(stderr, "[templr:error:data]") {
		t.Errorf("expected error kind 'data' in dir mode, stderr=%s", stderr)
	}
}

func TestExitCodes_ErrorMessageFormat(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name         string
		args         []string
		setup        func(string) error
		wantExitCode int
		wantPrefix   string
	}{
		{
			name:         "general error format",
			args:         []string{"-walk"},
			wantExitCode: 1,
			wantPrefix:   "[templr:error:args]",
		},
		{
			name: "data error format",
			setup: func(td string) error {
				in := filepath.Join(td, "in.tpl")
				return os.WriteFile(in, []byte("test"), 0o644)
			},
			args:         []string{"-in", "{{TD}}/in.tpl", "-data", "nonexistent.yaml"},
			wantExitCode: 3,
			wantPrefix:   "[templr:error:data]",
		},
		{
			name: "template error format",
			setup: func(td string) error {
				in := filepath.Join(td, "in.tpl")
				return os.WriteFile(in, []byte("{{ bad"), 0o644)
			},
			args:         []string{"-in", "{{TD}}/in.tpl"},
			wantExitCode: 2,
			wantPrefix:   "[templr:error:parse]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := t.TempDir()

			if tt.setup != nil {
				if err := tt.setup(td); err != nil {
					t.Fatal(err)
				}
			}

			// Replace {{TD}} placeholder with actual temp dir
			args := make([]string, len(tt.args))
			for i, arg := range tt.args {
				args[i] = strings.ReplaceAll(arg, "{{TD}}", td)
			}

			_, stderr, err := run(t, bin, args...)
			exitCode := getExitCode(err)

			if exitCode != tt.wantExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, exitCode)
			}

			if !strings.HasPrefix(strings.TrimSpace(stderr), tt.wantPrefix) {
				t.Errorf("expected stderr to start with %q, got: %s", tt.wantPrefix, stderr)
			}
		})
	}
}
