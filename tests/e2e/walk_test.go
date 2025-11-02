package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkPruneEmpty(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	// Make tree
	if err := os.MkdirAll(filepath.Join(src, "example"), 0o755); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(src, "example2"), 0o755); err != nil { t.Fatal(err) }

	// One real file, one empty render, one absent render
	if err := os.WriteFile(filepath.Join(src, "example", "test.tpl"), []byte("hello: world\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(src, "example", "test2.tpl"), []byte("{{- if .false }}never{{ end -}}\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(src, "example2", "test3.tpl"), []byte("{{- if .missing }}nope{{ end -}}\n"), 0o644); err != nil { t.Fatal(err) }

	_, stderr, err := run(t, bin, "--walk", "--src", src, "--dst", dst)
	if err != nil {
		t.Fatalf("templr walk failed: %v, stderr=%s", err, stderr)
	}

	// Assert outputs
	if _, err := os.Stat(filepath.Join(dst, "example", "test")); err != nil {
		t.Fatalf("expected example/test to be rendered: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "example", "test2")); err == nil {
		t.Fatalf("did not expect example/test2 file to exist")
	}
	if _, err := os.Stat(filepath.Join(dst, "example2")); err == nil {
		t.Fatalf("did not expect example2 dir to exist (should be pruned)")
	}
}
