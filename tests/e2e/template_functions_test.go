package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHumanizeFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "humanizeBytes",
			template: `{{ 1234567 | humanizeBytes }}`,
			expected: "1.2 MB",
		},
		{
			name:     "humanizeNumber",
			template: `{{ 1234567 | humanizeNumber }}`,
			expected: "1,234,567",
		},
		{
			name:     "ordinal",
			template: `{{ 1 | ordinal }}, {{ 2 | ordinal }}, {{ 3 | ordinal }}, {{ 21 | ordinal }}`,
			expected: "1st, 2nd, 3rd, 21st",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tplFile := filepath.Join(tmpDir, "test.tpl")
			outFile := filepath.Join(tmpDir, "out.txt")

			if err := os.WriteFile(tplFile, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}

			_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

			result, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}

			got := strings.TrimSpace(string(result))
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestTOMLFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	// Test toToml
	t.Run("toToml", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "toml.tpl")
		outFile := filepath.Join(tmpDir, "toml.txt")

		template := `{{- $data := dict "name" "test" "count" 42 }}
{{ $data | toToml }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "name = 'test'") && !strings.Contains(got, `name = "test"`) {
			t.Errorf("Expected TOML to contain name = 'test', got %q", got)
		}
		if !strings.Contains(got, "count = 42") {
			t.Errorf("Expected TOML to contain count = 42, got %q", got)
		}
	})

	// Test fromToml
	t.Run("fromToml", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "fromtoml.tpl")
		outFile := filepath.Join(tmpDir, "fromtoml.txt")

		template := `{{- $toml := "name = 'test'\ncount = 42" }}
{{- $data := fromToml $toml }}
name: {{ $data.name }}
count: {{ $data.count }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "name: test") {
			t.Errorf("Expected output to contain 'name: test', got %q", got)
		}
		if !strings.Contains(got, "count: 42") {
			t.Errorf("Expected output to contain 'count: 42', got %q", got)
		}
	})
}

func TestPathFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "pathExt",
			template: `{{ pathExt "file.txt" }}`,
			expected: ".txt",
		},
		{
			name:     "pathStem",
			template: `{{ pathStem "document.pdf" }}`,
			expected: "document",
		},
		{
			name:     "pathStem_complex",
			template: `{{ pathStem "/path/to/file.tar.gz" }}`,
			expected: "file.tar",
		},
		{
			name:     "pathNormalize",
			template: `{{ pathNormalize "a/b/../c" }}`,
			expected: "a/c",
		},
		{
			name:     "mimeType_png",
			template: `{{ mimeType "image.png" }}`,
			expected: "image/png",
		},
		{
			name:     "mimeType_json",
			template: `{{ mimeType "data.json" }}`,
			expected: "application/json",
		},
		{
			name:     "mimeType_unknown",
			template: `{{ mimeType "file.xyz" }}`,
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tplFile := filepath.Join(tmpDir, "test.tpl")
			outFile := filepath.Join(tmpDir, "out.txt")

			if err := os.WriteFile(tplFile, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}

			_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

			result, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}

			got := strings.TrimSpace(string(result))
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestValidationFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "isEmail_valid",
			template: `{{ isEmail "user@example.com" }}`,
			expected: "true",
		},
		{
			name:     "isEmail_invalid",
			template: `{{ isEmail "not-an-email" }}`,
			expected: "false",
		},
		{
			name:     "isURL_valid",
			template: `{{ isURL "https://example.com/path" }}`,
			expected: "true",
		},
		{
			name:     "isURL_invalid",
			template: `{{ isURL "not-a-url" }}`,
			expected: "false",
		},
		{
			name:     "isIPv4_valid",
			template: `{{ isIPv4 "192.168.1.1" }}`,
			expected: "true",
		},
		{
			name:     "isIPv4_invalid",
			template: `{{ isIPv4 "not-an-ip" }}`,
			expected: "false",
		},
		{
			name:     "isIPv4_ipv6",
			template: `{{ isIPv4 "2001:db8::1" }}`,
			expected: "false",
		},
		{
			name:     "isIPv6_valid",
			template: `{{ isIPv6 "2001:db8::1" }}`,
			expected: "true",
		},
		{
			name:     "isIPv6_invalid",
			template: `{{ isIPv6 "192.168.1.1" }}`,
			expected: "false",
		},
		{
			name:     "isUUID_valid",
			template: `{{ isUUID "550e8400-e29b-41d4-a716-446655440000" }}`,
			expected: "true",
		},
		{
			name:     "isUUID_invalid",
			template: `{{ isUUID "not-a-uuid" }}`,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tplFile := filepath.Join(tmpDir, "test.tpl")
			outFile := filepath.Join(tmpDir, "out.txt")

			if err := os.WriteFile(tplFile, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}

			_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

			result, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}

			got := strings.TrimSpace(string(result))
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestValidationInRealScenario(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()

	valuesFile := filepath.Join(tmpDir, "values.yaml")
	valuesContent := `
email: admin@example.com
website: https://example.com
ipv4: 10.0.0.1
ipv6: 2001:db8::1
requestId: 550e8400-e29b-41d4-a716-446655440000
`
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tplFile := filepath.Join(tmpDir, "validation.tpl")
	template := `{{- if not (isEmail .email) }}
ERROR: Invalid email
{{- end }}
{{- if not (isURL .website) }}
ERROR: Invalid URL
{{- end }}
{{- if not (isIPv4 .ipv4) }}
ERROR: Invalid IPv4
{{- end }}
{{- if not (isIPv6 .ipv6) }}
ERROR: Invalid IPv6
{{- end }}
{{- if not (isUUID .requestId) }}
ERROR: Invalid UUID
{{- end }}
All validations passed!`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "output.txt")
	_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--inject-guard=false")

	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(result))
	if strings.Contains(got, "ERROR") {
		t.Errorf("Validation should have passed but got errors: %q", got)
	}
	if !strings.Contains(got, "All validations passed!") {
		t.Errorf("Expected 'All validations passed!' in output, got %q", got)
	}
}
