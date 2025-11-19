package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// repoRoot walks up from startDir until it finds a go.mod, and returns that directory.
func repoRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return startDir
		}
		dir = parent
	}
}

// buildTemplr builds the templr binary from the repository root and returns its path.
func buildTemplr(t *testing.T, startDir string) string {
	t.Helper()
	root := repoRoot(startDir)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	bin := filepath.Join(root, "templr-testbin")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed in %s: %v\n%s", root, err, string(out))
	}
	return bin
}

func run(t *testing.T, bin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// normalizeOut canonicalizes guard markers and trims trailing blank lines.
func normalizeOut(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && strings.Contains(lines[0], "templr generated") {
		// Canonicalize any variant like "# #templr generated", "<!-- #templr generated -->", etc.
		lines[0] = "#templr generated"
	}
	// Drop trailing blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
