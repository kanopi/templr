package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtFlagInWalkMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("Title: {{ .title | default \"Docs\" }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "values.yaml"), []byte("title: Handbook\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "--walk", "--src", src, "--dst", dst, "--ext", "md")
	if err != nil {
		t.Fatalf("templr walk failed: %v, stderr=%s", err, stderr)
	}
	if _, err := os.Stat(filepath.Join(dst, "README")); err != nil {
		t.Fatalf("expected rendered README without extension: %v", err)
	}
}
