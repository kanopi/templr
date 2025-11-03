package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeHelperOverridesGlobalDefaultMissing(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	in := filepath.Join(td, "tpl.tpl")
	out := filepath.Join(td, "out.txt")

	// Use safe helper to provide per-variable fallback, even if a global default is set.
	tpl := "" +
		"User: {{ safe .user \"anon\" }}\n" + // safe should output "anon"
		"Team: {{ .team }}\n" // missing -> replaced by global
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
	want := normalizeOut("#templr generated\nUser: anon\nTeam: N/A\n")
	if got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
