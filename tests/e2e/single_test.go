package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSingleFileRender(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := `app: {{ .app }}
replicas: {{ .replicas | default 1 }}`
	vals := "app: kanopi\nreplicas: 2\n"
	in := filepath.Join(td, "in.tpl")
	data := filepath.Join(td, "values.yaml")
	out := filepath.Join(td, "out.yaml")
	if err := os.WriteFile(in, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte(vals), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := run(t, bin, "-in", in, "-data", data, "-out", out)
	if err != nil {
		t.Fatalf("templr failed: %v, stderr=%s", err, stderr)
	}
	gotBytes, _ := os.ReadFile(out)
	got := normalizeOut(string(gotBytes))
	want := normalizeOut("#templr generated\napp: kanopi\nreplicas: 2\n")
	if got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
