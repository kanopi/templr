package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildTemplr(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "templr")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Dir = filepath.Clean(".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return bin
}

func TestSingleFile(t *testing.T) {
	bin := buildTemplr(t)
	out := filepath.Join(t.TempDir(), "out.yaml")
	cmd := exec.Command(bin, "-in", "playground/template.tpl", "-data", "playground/values.yaml", "-out", out)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, outb)
	}
	got, _ := os.ReadFile(out)
	want, _ := os.ReadFile("tests/golden/single/out.yaml")
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch:\nGOT:\n%s\nWANT:\n%s", got, want)
	}
}

func TestGuard(t *testing.T) {
	bin := buildTemplr(t)
	tmp := t.TempDir()
	target := filepath.Join(tmp, "file.yaml")

	// No guard present -> skip overwrite
	if err := os.WriteFile(target, []byte("content: original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-in", "playground/guard/tpl.tpl", "-out", target)
	_ = cmd.Run()
	got, _ := os.ReadFile(target)
	if string(got) != "content: original\n" {
		t.Fatalf("should have skipped overwrite when guard missing")
	}

	// Add guard -> overwrite succeeds
	if err := os.WriteFile(target, []byte("#templr generated\ncontent: original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(bin, "-in", "playground/guard/tpl.tpl", "-out", target)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, outb)
	}
	got, _ = os.ReadFile(target)
	if !bytes.Contains(got, []byte("content: updated")) {
		t.Fatalf("expected updated content with guard present")
	}
}
