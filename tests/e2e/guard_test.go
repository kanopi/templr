package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardOverwriteBehavior(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "in.tpl")
	out := filepath.Join(td, "out.yaml")

	if err := os.WriteFile(in, []byte("content: updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Case 1: Existing file WITHOUT marker -> should not overwrite
	if err := os.WriteFile(out, []byte("content: original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, _ = run(t, bin, "-in", in, "-out", out, "--guard", "#templr generated")
	got1, _ := os.ReadFile(out)
	if strings.Contains(string(got1), "updated") {
		t.Fatalf("expected skip overwrite when guard marker missing; got=%q", string(got1))
	}

	// Case 2: Existing file WITH marker -> should overwrite
	if err := os.WriteFile(out, []byte("#templr generated\ncontent: original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := run(t, bin, "-in", in, "-out", out, "--guard", "#templr generated")
	if err != nil {
		t.Fatalf("templr run failed: %v", err)
	}
	got2, _ := os.ReadFile(out)
	if !strings.Contains(string(got2), "updated") {
		t.Fatalf("expected overwrite when guard marker present; got=%q", string(got2))
	}
}
