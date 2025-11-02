package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultMissingFlag(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "tpl.tpl")
	out := filepath.Join(td, "out.txt")

	// Template references two missing keys and one with a local default via Sprig.
	// The missing keys should use --default-missing replacement; the local default remains intact.
	tpl := "" +
		"Name: {{ .name }}\n" + // missing -> replaced
		"Role: {{ .role | default \"dev\" }}\n" + // has per-field default
		"City: {{ .city }}\n" // missing -> replaced
	if err := os.WriteFile(in, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "--default-missing", "N/A", "-out", out)
	if err != nil {
		t.Fatalf("templr failed: %v, stderr=%s", err, stderr)
	}

	gotBytes, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeOut(string(gotBytes))
	want := normalizeOut("#templr generated\nName: N/A\nRole: dev\nCity: N/A\n")
	if got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
