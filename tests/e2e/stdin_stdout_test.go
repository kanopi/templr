package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdinStdoutRender(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	vals := filepath.Join(td, "values.yaml")
	if err := os.WriteFile(vals, []byte("name: templr\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "-data", vals)
	cmd.Stdin = bytes.NewBufferString("Hello {{ .name }}\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // acceptable for this simple smoke test
	if err := cmd.Run(); err != nil {
		t.Fatalf("templr run failed: %v\n%s", err, out.String())
	}

	got := normalizeOut(out.String())
	if !strings.Contains(got, "Hello templr") {
		t.Fatalf("expected stdout to contain rendered content, got:\n%s", got)
	}
}
