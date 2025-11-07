package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//nolint:dupl // Test patterns are intentionally similar
func TestBase64URLFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "base64url_encode",
			template: `{{ "hello world" | base64url }}`,
			expected: "aGVsbG8gd29ybGQ=",
		},
		{
			name:     "base64url_roundtrip",
			template: `{{ "test data" | base64url | base64urlDecode }}`,
			expected: "test data",
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

//nolint:dupl // Test patterns are intentionally similar
func TestBase32Functions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "base32_encode",
			template: `{{ "hello" | base32 }}`,
			expected: "NBSWY3DP",
		},
		{
			name:     "base32_roundtrip",
			template: `{{ "test" | base32 | base32Decode }}`,
			expected: "test",
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

func TestCSVFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	t.Run("fromCsv", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "test.tpl")
		outFile := filepath.Join(tmpDir, "out.txt")

		template := "{{- $csv := `name,age,city\nJohn,30,NYC\nJane,25,LA` }}\n" +
			"{{- $data := fromCsv $csv }}\n" +
			"{{- range $data }}\n" +
			"{{ .name }}: {{ .age }} ({{ .city }})\n" +
			"{{- end }}"

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "John: 30 (NYC)") {
			t.Errorf("Expected output to contain 'John: 30 (NYC)', got %q", got)
		}
		if !strings.Contains(got, "Jane: 25 (LA)") {
			t.Errorf("Expected output to contain 'Jane: 25 (LA)', got %q", got)
		}
	})

	t.Run("csvColumn", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "test.tpl")
		outFile := filepath.Join(tmpDir, "out.txt")

		template := "{{- $csv := `name,age,city\nJohn,30,NYC\nJane,25,LA` }}\n" +
			"{{- $names := csvColumn $csv \"name\" }}\n" +
			"{{- range $names }}\n" +
			"- {{ . }}\n" +
			"{{- end }}"

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "- John") {
			t.Errorf("Expected output to contain '- John', got %q", got)
		}
		if !strings.Contains(got, "- Jane") {
			t.Errorf("Expected output to contain '- Jane', got %q", got)
		}
	})
}

//nolint:dupl // Test patterns are intentionally similar
func TestNetworkFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "cidrContains_true",
			template: `{{ cidrContains "10.0.0.5" "10.0.0.0/24" }}`,
			expected: "true",
		},
		{
			name:     "cidrContains_false",
			template: `{{ cidrContains "10.0.1.5" "10.0.0.0/24" }}`,
			expected: "false",
		},
		{
			name:     "ipVersion_v4",
			template: `{{ ipVersion "192.168.1.1" }}`,
			expected: "4",
		},
		{
			name:     "ipVersion_v6",
			template: `{{ ipVersion "2001:db8::1" }}`,
			expected: "6",
		},
		{
			name:     "ipPrivate_true",
			template: `{{ ipPrivate "192.168.1.1" }}`,
			expected: "true",
		},
		{
			name:     "ipPrivate_false",
			template: `{{ ipPrivate "8.8.8.8" }}`,
			expected: "false",
		},
		{
			name:     "ipAdd",
			template: `{{ ipAdd "10.0.0.1" 5 }}`,
			expected: "10.0.0.6",
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

func TestCIDRHosts(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	tplFile := filepath.Join(tmpDir, "test.tpl")
	outFile := filepath.Join(tmpDir, "out.txt")

	template := `{{- $hosts := cidrHosts "10.0.0.0/30" }}
{{- range $hosts }}
{{ . }}
{{- end }}`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := string(result)
	// /30 should give us 2 usable hosts (network and broadcast excluded)
	if !strings.Contains(got, "10.0.0.1") {
		t.Errorf("Expected output to contain '10.0.0.1', got %q", got)
	}
	if !strings.Contains(got, "10.0.0.2") {
		t.Errorf("Expected output to contain '10.0.0.2', got %q", got)
	}
	// Should NOT contain network or broadcast
	if strings.Contains(got, "10.0.0.0") {
		t.Errorf("Should not contain network address '10.0.0.0', got %q", got)
	}
	if strings.Contains(got, "10.0.0.3") {
		t.Errorf("Should not contain broadcast address '10.0.0.3', got %q", got)
	}
}

//nolint:dupl // Test patterns are intentionally similar
func TestMathFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "sum",
			template: `{{ sum (list 1 2 3 4 5) }}`,
			expected: "15",
		},
		{
			name:     "avg",
			template: `{{ avg (list 2 4 6 8) }}`,
			expected: "5",
		},
		{
			name:     "median_odd",
			template: `{{ median (list 1 2 3 4 5) }}`,
			expected: "3",
		},
		{
			name:     "clamp_below",
			template: `{{ clamp -5 0 10 }}`,
			expected: "0",
		},
		{
			name:     "clamp_above",
			template: `{{ clamp 15 0 10 }}`,
			expected: "10",
		},
		{
			name:     "clamp_within",
			template: `{{ clamp 5 0 10 }}`,
			expected: "5",
		},
		{
			name:     "roundTo",
			template: `{{ roundTo 3.14159 2 }}`,
			expected: "3.14",
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

func TestStddevAndPercentile(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	t.Run("stddev", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "test.tpl")
		outFile := filepath.Join(tmpDir, "out.txt")

		template := `{{ stddev (list 2 4 4 4 5 5 7 9) | roundTo 2 }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := strings.TrimSpace(string(result))
		// Standard deviation of this dataset is approximately 2
		expected := "2"
		if got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}
	})

	t.Run("percentile", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "test.tpl")
		outFile := filepath.Join(tmpDir, "out.txt")

		template := `{{ percentile (list 1 2 3 4 5 6 7 8 9 10) 90 }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := strings.TrimSpace(string(result))
		// 90th percentile of 1-10 is 9
		expected := "9"
		if got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}
	})
}

func TestTier2IntegrationScenario(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	valuesContent := `
serverData: |
  hostname,ip,cpu,memory
  web1,10.0.0.10,2,4096
  web2,10.0.0.11,4,8192
  web3,10.0.0.12,2,4096

network:
  cidr: "10.0.0.0/24"
  gateway: "10.0.0.1"
`
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tplFile := filepath.Join(tmpDir, "template.tpl")
	template := `# Server Configuration
{{- $servers := fromCsv .serverData }}

## Servers
{{- range $servers }}
- Hostname: {{ .hostname }}
  IP: {{ .ip }}
  {{- if cidrContains .ip $.network.cidr }}
  Network: In range ✓
  {{- end }}
  CPU Cores: {{ .cpu }}
  Memory: {{ .memory | atoi | humanizeBytes }}
{{- end }}

## Statistics
{{- $cpus := csvColumn .serverData "cpu" }}
{{- $cpuNums := list }}
{{- range $cpus }}
  {{- $cpuNums = append $cpuNums (. | atoi) }}
{{- end }}
Total CPU Cores: {{ sum $cpuNums }}
Average CPU: {{ avg $cpuNums }}

## Network
Gateway: {{ .network.gateway }}
Next IP: {{ ipAdd .network.gateway 1 }}
Private: {{ ipPrivate .network.gateway }}
`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "output.txt")
	_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--inject-guard=false")

	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := string(result)

	// Verify CSV parsing worked
	if !strings.Contains(got, "Hostname: web1") {
		t.Errorf("Expected CSV parsing, got %q", got)
	}

	// Verify network functions
	if !strings.Contains(got, "In range ✓") {
		t.Errorf("Expected CIDR check to pass, got %q", got)
	}

	// Verify math functions
	if !strings.Contains(got, "Total CPU Cores: 8") {
		t.Errorf("Expected sum to work, got %q", got)
	}

	// Verify humanize
	if !strings.Contains(got, "4.1 kB") || !strings.Contains(got, "8.2 kB") {
		t.Errorf("Expected humanizeBytes to work, got %q", got)
	}

	// Verify IP operations
	if !strings.Contains(got, "Next IP: 10.0.0.2") {
		t.Errorf("Expected ipAdd to work, got %q", got)
	}
}
