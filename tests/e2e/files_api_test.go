package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesAPIGet(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	dir := filepath.Join(td, "dir")
	if err := os.MkdirAll(filepath.Join(dir, "certs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "certs", "tls.crt"), []byte("-----BEGIN CERT-----\nABC123\n-----END CERT-----\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	in := filepath.Join(dir, "secret.tpl")
	if err := os.WriteFile(in, []byte("kind: Secret\ndata:\n  cert: |\n{{ (.Files.Get \"certs/tls.crt\") | trim | indent 4 }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(td, "out.yaml")
	_, stderr, err := run(t, bin, "-dir", dir, "-in", in, "-out", out)
	if err != nil {
		t.Fatalf("templr dir failed: %v, stderr=%s", err, stderr)
	}

	gotBytes, _ := os.ReadFile(out)
	got := normalizeOut(string(gotBytes))
	want := normalizeOut("#templr generated\nkind: Secret\ndata:\n  cert: |\n    -----BEGIN CERT-----\n    ABC123\n    -----END CERT-----\n")
	if got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
