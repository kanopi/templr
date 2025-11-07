package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kanopi/templr/internal/app"
)

func TestFilesAPI_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}

	if !api.Exists("test.txt") {
		t.Error("Exists should return true for existing file")
	}

	if api.Exists("missing.txt") {
		t.Error("Exists should return false for missing file")
	}
}

func TestFilesAPI_Stat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	info, err := api.Stat("test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", info.Name)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), info.Size)
	}

	if info.IsDir {
		t.Error("Expected IsDir to be false")
	}

	// Test directory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dirInfo, err := api.Stat("subdir")
	if err != nil {
		t.Fatalf("Stat on directory failed: %v", err)
	}

	if !dirInfo.IsDir {
		t.Error("Expected IsDir to be true for directory")
	}
}

func TestFilesAPI_Lines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lines.txt")
	content := "line1\nline2\nline3"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	lines, err := api.Lines("lines.txt")
	if err != nil {
		t.Fatalf("Lines failed: %v", err)
	}

	expected := []string{"line1", "line2", "line3"}
	if len(lines) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(lines))
	}

	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected[i], line)
		}
	}

	// Test file with trailing newline
	testFile2 := filepath.Join(tmpDir, "trailing.txt")
	contentTrailing := "line1\nline2\n"
	if err := os.WriteFile(testFile2, []byte(contentTrailing), 0o644); err != nil {
		t.Fatal(err)
	}

	lines2, err := api.Lines("trailing.txt")
	if err != nil {
		t.Fatalf("Lines failed: %v", err)
	}

	// Should get 2 lines (trailing newline removed)
	if len(lines2) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines2))
	}
}

func TestFilesAPI_ReadDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	entries, err := api.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// Should have 3 files + 1 directory
	if len(entries) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(entries))
	}

	// Check if all files are present
	found := make(map[string]bool)
	for _, entry := range entries {
		found[entry] = true
	}

	for _, f := range files {
		if !found[f] {
			t.Errorf("File %s not found in ReadDir results", f)
		}
	}

	if !found["subdir"] {
		t.Error("Subdirectory not found in ReadDir results")
	}
}

func TestFilesAPI_GlobDetails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"test1.txt": "content1",
		"test2.txt": "content2",
		"other.md":  "markdown",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	api := app.FilesAPI{Root: tmpDir}
	infos, err := api.GlobDetails("*.txt")
	if err != nil {
		t.Fatalf("GlobDetails failed: %v", err)
	}

	// Should match 2 .txt files
	if len(infos) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(infos))
	}

	for _, info := range infos {
		if info.Name != "test1.txt" && info.Name != "test2.txt" {
			t.Errorf("Unexpected file name: %s", info.Name)
		}

		if info.Size == 0 {
			t.Error("Expected non-zero file size")
		}

		if info.IsDir {
			t.Error("Expected IsDir to be false")
		}
	}
}

func TestFilesAPI_AsBase64(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.bin")
	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	encoded, err := api.AsBase64("data.bin")
	if err != nil {
		t.Fatalf("AsBase64 failed: %v", err)
	}

	expected := "SGVsbG8sIFdvcmxkIQ=="
	if encoded != expected {
		t.Errorf("Expected '%s', got '%s'", expected, encoded)
	}
}

func TestFilesAPI_AsHex(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.bin")
	content := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	encoded, err := api.AsHex("data.bin")
	if err != nil {
		t.Fatalf("AsHex failed: %v", err)
	}

	expected := "deadbeef"
	if encoded != expected {
		t.Errorf("Expected '%s', got '%s'", expected, encoded)
	}
}

func TestFilesAPI_AsDataURL(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}

	// Test with auto-detect MIME type
	dataURL, err := api.AsDataURL("test.txt", "")
	if err != nil {
		t.Fatalf("AsDataURL failed: %v", err)
	}

	// Should default to application/octet-stream for .txt
	expectedPrefix := "data:application/octet-stream;base64,"
	if len(dataURL) < len(expectedPrefix) || dataURL[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected data URL to start with '%s'", expectedPrefix)
	}

	// Test with explicit MIME type
	dataURL2, err := api.AsDataURL("test.txt", "text/plain")
	if err != nil {
		t.Fatalf("AsDataURL with explicit MIME failed: %v", err)
	}

	expectedPrefix2 := "data:text/plain;base64,"
	if len(dataURL2) < len(expectedPrefix2) || dataURL2[:len(expectedPrefix2)] != expectedPrefix2 {
		t.Errorf("Expected data URL to start with '%s'", expectedPrefix2)
	}
}

func TestFilesAPI_AsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.json")
	content := `{"name": "test", "value": 123, "active": true}`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	data, err := api.AsJSON("data.json")
	if err != nil {
		t.Fatalf("AsJSON failed: %v", err)
	}

	if data["name"] != "test" {
		t.Errorf("Expected name='test', got '%v'", data["name"])
	}

	if data["value"] != float64(123) { // JSON numbers are float64
		t.Errorf("Expected value=123, got '%v'", data["value"])
	}

	if data["active"] != true {
		t.Errorf("Expected active=true, got '%v'", data["active"])
	}

	// Test invalid JSON
	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = api.AsJSON("invalid.json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestFilesAPI_AsYAML(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.yaml")
	content := `name: test
value: 123
active: true
nested:
  key: value`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}
	data, err := api.AsYAML("data.yaml")
	if err != nil {
		t.Fatalf("AsYAML failed: %v", err)
	}

	if data["name"] != "test" {
		t.Errorf("Expected name='test', got '%v'", data["name"])
	}

	if data["value"] != 123 {
		t.Errorf("Expected value=123, got '%v'", data["value"])
	}

	if data["active"] != true {
		t.Errorf("Expected active=true, got '%v'", data["active"])
	}

	// Check nested
	nested, ok := data["nested"].(map[string]any)
	if !ok {
		t.Fatal("Expected nested to be a map")
	}

	if nested["key"] != "value" {
		t.Errorf("Expected nested.key='value', got '%v'", nested["key"])
	}

	// Test invalid YAML
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidFile, []byte(":\ninvalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = api.AsYAML("invalid.yaml")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestFilesAPI_AsLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lines.txt")
	content := "line1\nline2\nline3"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	api := app.FilesAPI{Root: tmpDir}

	// AsLines should be an alias for Lines
	lines1, err1 := api.AsLines("lines.txt")
	lines2, err2 := api.Lines("lines.txt")

	if err1 != nil || err2 != nil {
		t.Fatal("Expected no errors")
	}

	if len(lines1) != len(lines2) {
		t.Error("AsLines and Lines should return the same results")
	}

	for i := range lines1 {
		if lines1[i] != lines2[i] {
			t.Error("AsLines and Lines should return identical content")
		}
	}
}

func TestFilesAPI_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	api := app.FilesAPI{Root: tmpDir}

	// Test Stat on non-existent file
	_, err := api.Stat("missing.txt")
	if err == nil {
		t.Error("Expected error for Stat on missing file")
	}

	// Test Lines on non-existent file
	_, err = api.Lines("missing.txt")
	if err == nil {
		t.Error("Expected error for Lines on missing file")
	}

	// Test ReadDir on non-existent directory
	_, err = api.ReadDir("missing")
	if err == nil {
		t.Error("Expected error for ReadDir on missing directory")
	}

	// Test AsBase64 on non-existent file
	_, err = api.AsBase64("missing.txt")
	if err == nil {
		t.Error("Expected error for AsBase64 on missing file")
	}

	// Test AsJSON on non-existent file
	_, err = api.AsJSON("missing.json")
	if err == nil {
		t.Error("Expected error for AsJSON on missing file")
	}

	// Test AsYAML on non-existent file
	_, err = api.AsYAML("missing.yaml")
	if err == nil {
		t.Error("Expected error for AsYAML on missing file")
	}
}
