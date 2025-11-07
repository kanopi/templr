package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMimeType(t *testing.T) {
	// Test via AsDataURL which uses detectMimeType internally
	tests := []struct {
		filename       string
		expectedPrefix string
	}{
		{"test.png", "data:image/png;base64,"},
		{"photo.jpg", "data:image/jpeg;base64,"},
		{"icon.svg", "data:image/svg+xml;base64,"},
		{"style.css", "data:text/css;base64,"},
		{"script.js", "data:application/javascript;base64,"},
		{"data.json", "data:application/json;base64,"},
		{"page.html", "data:text/html;base64,"},
		{"doc.pdf", "data:application/pdf;base64,"},
		{"unknown.xyz", "data:application/octet-stream;base64,"},
	}

	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Each test gets its own tmpDir
			tmpDir := t.TempDir()

			// Create test file
			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
				t.Fatal(err)
			}

			tpl := filepath.Join(tmpDir, "test.tpl")
			out := filepath.Join(tmpDir, "out.txt")

			tplContent := "{{ .Files.AsDataURL \"" + tt.filename + "\" \"\" }}"
			if err := os.WriteFile(tpl, []byte(tplContent), 0o644); err != nil {
				t.Fatal(err)
			}

			_, _, _ = run(t, bin, "render", "-i", tpl, "-o", out, "--inject-guard=false")

			result, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}

			resultStr := string(result)
			if len(resultStr) < len(tt.expectedPrefix) || resultStr[:len(tt.expectedPrefix)] != tt.expectedPrefix {
				t.Errorf("Expected data URL to start with %q, got %q", tt.expectedPrefix, resultStr)
			}
		})
	}
}
