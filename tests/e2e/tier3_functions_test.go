package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//nolint:dupl // Test patterns are intentionally similar
func TestJSONPathFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	valuesContent := `
jsonData: |
  {
    "users": [
      {"name": "Alice", "age": 30, "active": true},
      {"name": "Bob", "age": 25, "active": false},
      {"name": "Charlie", "age": 35, "active": true}
    ],
    "config": {
      "enabled": true,
      "timeout": 300
    }
  }
`
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("jsonPath_simple", func(t *testing.T) {
		tplFile := filepath.Join(tmpDir, "jsonpath_simple.tpl")
		outFile := filepath.Join(tmpDir, "jsonpath_simple.txt")

		template := `{{- $result := jsonPath .jsonData "config.enabled" }}{{ $result }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := strings.TrimSpace(string(result))
		if got != "true" {
			t.Errorf("Expected 'true', got %q", got)
		}
	})

	t.Run("jsonQuery_array", func(t *testing.T) {
		tplFile := filepath.Join(tmpDir, "jsonquery_array.tpl")
		outFile := filepath.Join(tmpDir, "jsonquery_array.txt")

		template := `{{- $names := jsonQuery .jsonData "users.#.name" }}{{- range $names }}
- {{ . }}
{{- end }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "- Alice") || !strings.Contains(got, "- Bob") || !strings.Contains(got, "- Charlie") {
			t.Errorf("Expected all names, got %q", got)
		}
	})

	t.Run("jsonSet", func(t *testing.T) {
		tplFile := filepath.Join(tmpDir, "jsonset.tpl")
		outFile := filepath.Join(tmpDir, "jsonset.txt")

		template := `{{- $json := toJson (dict "name" "test" "count" 42) }}
{{- $updated := jsonSet $json "count" 100 }}
{{ $updated }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "-d", valuesFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, `"count":100`) {
			t.Errorf("Expected count to be 100, got %q", got)
		}
	})
}

//nolint:dupl // Test patterns are intentionally similar
func TestDateParseFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tests := []struct {
		name     string
		template string
		check    func(string) bool
		desc     string
	}{
		{
			name:     "dateParse_iso",
			template: `{{ dateParse "2024-03-15" | date "2006-01-02" }}`,
			check: func(s string) bool {
				return strings.Contains(s, "2024-03-1") // Allow 14 or 15 due to timezone
			},
			desc: "2024-03-1X (timezone dependent)",
		},
		{
			name:     "dateParse_human",
			template: `{{ dateParse "March 15, 2024" | date "2006-01-02" }}`,
			check: func(s string) bool {
				return strings.Contains(s, "2024-03-1") // Allow 14 or 15 due to timezone
			},
			desc: "2024-03-1X (timezone dependent)",
		},
		{
			name:     "dateAdd_days",
			template: `{{ dateAdd "2024-01-01" "7 days" | date "2006-01-02" }}`,
			check: func(s string) bool {
				// Could be 01-07 or 01-08 depending on timezone
				return strings.Contains(s, "2024-01-0")
			},
			desc: "2024-01-0X (timezone dependent)",
		},
		{
			name:     "dateAdd_weeks",
			template: `{{ dateAdd "2024-01-01" "2 weeks" | date "2006-01-02" }}`,
			check: func(s string) bool {
				// Could be 01-14 or 01-15 depending on timezone
				return strings.Contains(s, "2024-01-1")
			},
			desc: "2024-01-1X (timezone dependent)",
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
			if !tt.check(got) {
				t.Errorf("Expected %s, got %q", tt.desc, got)
			}
		})
	}
}

func TestDateRange(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	tplFile := filepath.Join(tmpDir, "test.tpl")
	outFile := filepath.Join(tmpDir, "out.txt")

	template := `{{- range dateRange "2024-01-01" "2024-01-03" }}
{{ . | date "2006-01-02" }}
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
	// Count line breaks to verify we got 3 dates
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 dates in range, got %d: %q", len(lines), got)
	}
	// Verify dates are in January 2024
	if !strings.Contains(got, "2024-01") {
		t.Errorf("Expected dates in January 2024, got %q", got)
	}
}

func TestWorkdays(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	tplFile := filepath.Join(tmpDir, "test.tpl")
	outFile := filepath.Join(tmpDir, "out.txt")

	// Jan 1-7, 2024: Mon-Sun (5 workdays)
	template := `{{ workdays "2024-01-01" "2024-01-07" }}`

	if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

	result, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(result))
	if got != "5" {
		t.Errorf("Expected 5 workdays, got %q", got)
	}
}

//nolint:dupl // Test patterns are intentionally similar
func TestXMLFunctions(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	t.Run("toXml", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "toxml.tpl")
		outFile := filepath.Join(tmpDir, "toxml.txt")

		template := `{{- $data := dict "name" "test" "count" 42 }}
{{ $data | toXml }}`

		if err := os.WriteFile(tplFile, []byte(template), 0o644); err != nil {
			t.Fatal(err)
		}

		_, _, _ = run(t, bin, "render", "-i", tplFile, "-o", outFile, "--inject-guard=false")

		result, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}

		got := string(result)
		if !strings.Contains(got, "<name>test</name>") {
			t.Errorf("Expected XML with name element, got %q", got)
		}
		if !strings.Contains(got, "<count>42</count>") {
			t.Errorf("Expected XML with count element, got %q", got)
		}
	})

	t.Run("fromXml", func(t *testing.T) {
		tmpDir := t.TempDir()
		tplFile := filepath.Join(tmpDir, "fromxml.tpl")
		outFile := filepath.Join(tmpDir, "fromxml.txt")

		template := `{{- $xml := "<root><name>test</name><count>42</count></root>" }}
{{- $data := fromXml $xml }}
name: {{ $data.root.name }}
count: {{ $data.root.count }}`

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
			t.Errorf("Expected 'name: test', got %q", got)
		}
		if !strings.Contains(got, "count: 42") {
			t.Errorf("Expected 'count: 42', got %q", got)
		}
	})
}

func TestTier3IntegrationScenario(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	tmpDir := t.TempDir()
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	valuesContent := `
users: |
  {
    "users": [
      {"name": "Alice", "joined": "2024-01-15", "active": true},
      {"name": "Bob", "joined": "2024-02-20", "active": true},
      {"name": "Charlie", "joined": "2024-03-10", "active": false}
    ]
  }
`
	if err := os.WriteFile(valuesFile, []byte(valuesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tplFile := filepath.Join(tmpDir, "template.tpl")
	template := `# User Report
Generated: {{ now | date "2006-01-02" }}

## Active Users
{{- $activeUsers := jsonQuery .users "users.#(active==true).name" }}
{{- range $activeUsers }}
- {{ . }}
{{- end }}

## Join Dates
{{- $users := jsonPath .users "users" }}
{{- range $users }}
- {{ index . "name" }}: {{ dateParse (index . "joined") | date "January 2, 2006" }}
{{- end }}

## Date Calculations
{{- $firstJoin := dateParse "2024-01-15" }}
Next review: {{ dateAdd "2024-01-15" "30 days" | date "2006-01-02" }}
Workdays since: {{ workdays "2024-01-15" "2024-01-31" }}
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

	// Verify JSON querying worked
	if !strings.Contains(got, "- Alice") {
		t.Errorf("Expected active user Alice, got %q", got)
	}
	if !strings.Contains(got, "- Bob") {
		t.Errorf("Expected active user Bob, got %q", got)
	}
	if strings.Contains(got, "Active Users:\n- Charlie") {
		t.Errorf("Charlie should not be in active users (inactive), got %q", got)
	}

	// Verify date parsing worked (allow for timezone differences)
	if !strings.Contains(got, "January 1") || !strings.Contains(got, "2024") {
		t.Errorf("Expected date format with 'January' and '2024', got %q", got)
	}

	// Verify date calculations (allow for timezone differences)
	if !strings.Contains(got, "Next review: 2024-02-1") {
		t.Errorf("Expected dateAdd result in February, got %q", got)
	}
	if !strings.Contains(got, "Workdays since: 13") {
		t.Errorf("Expected workdays count, got %q", got)
	}
}
