package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteOptimizationUnchangedContent(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := filepath.Join(td, "test.tpl")
	out := filepath.Join(td, "output.txt")

	// Create template
	if err := os.WriteFile(tpl, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First render
	_, _, _ = run(t, bin, "-in", tpl, "-out", out)

	// Get initial mtime
	info1, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file should exist after first render: %v", err)
	}
	mtime1 := info1.ModTime()

	// Wait a bit to ensure mtime would differ if file was rewritten
	time.Sleep(10 * time.Millisecond)

	// Second render with same content
	_, _, _ = run(t, bin, "-in", tpl, "-out", out)

	// Get mtime after second render
	info2, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file should still exist: %v", err)
	}
	mtime2 := info2.ModTime()

	// mtime should NOT have changed (file was not rewritten)
	if !mtime1.Equal(mtime2) {
		t.Errorf("expected mtime to remain unchanged when content is identical. Before: %v, After: %v", mtime1, mtime2)
	}
}

func TestWriteOptimizationChangedContent(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl1 := filepath.Join(td, "test1.tpl")
	tpl2 := filepath.Join(td, "test2.tpl")
	out := filepath.Join(td, "output.txt")

	// Create first template
	if err := os.WriteFile(tpl1, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First render (with guard injection enabled by default)
	_, _, _ = run(t, bin, "-in", tpl1, "-out", out)

	info1, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file should exist after first render: %v", err)
	}
	mtime1 := info1.ModTime()

	// Wait to ensure mtime would differ
	time.Sleep(10 * time.Millisecond)

	// Create second template with different content (changed by 1 byte)
	// Include guard in template so the test works with guard checking
	if err := os.WriteFile(tpl2, []byte("Hello Worlx"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second render with different content
	_, _, _ = run(t, bin, "-in", tpl2, "-out", out)

	info2, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file should exist after second render: %v", err)
	}
	mtime2 := info2.ModTime()

	// mtime SHOULD have changed (file was rewritten)
	if mtime1.Equal(mtime2) || mtime2.Before(mtime1) {
		t.Errorf("expected mtime to be updated when content changed. Before: %v, After: %v", mtime1, mtime2)
	}

	// Verify content was actually updated
	content, _ := os.ReadFile(out)
	if !strings.Contains(string(content), "Hello Worlx") {
		t.Errorf("expected content to contain 'Hello Worlx', got: %q", string(content))
	}
}

func TestWriteOptimizationNewFile(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := filepath.Join(td, "test.tpl")
	out := filepath.Join(td, "output.txt")

	// Create template
	if err := os.WriteFile(tpl, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// File doesn't exist yet
	if _, err := os.Stat(out); err == nil {
		t.Fatal("output file should not exist before first render")
	}

	// First render (file doesn't exist)
	stdout1, _, _ := run(t, bin, "-in", tpl, "-out", out)

	// File should now exist
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output file should exist after first render: %v", err)
	}

	// Should have printed "rendered" message
	if !strings.Contains(stdout1, "rendered") {
		t.Errorf("expected stdout to contain 'rendered' for new file, got: %s", stdout1)
	}

	mtime1, _ := os.Stat(out)
	time.Sleep(10 * time.Millisecond)

	// Second render (file exists, same content)
	stdout2, _, _ := run(t, bin, "-in", tpl, "-out", out)

	mtime2, _ := os.Stat(out)

	// mtime should NOT change (file was not rewritten)
	if !mtime1.ModTime().Equal(mtime2.ModTime()) {
		t.Errorf("expected mtime to remain unchanged on second render with same content")
	}

	// Should NOT print "rendered" message (file was skipped)
	if strings.Contains(stdout2, "rendered") {
		t.Errorf("expected no 'rendered' message when content unchanged, got: %s", stdout2)
	}
}

func TestWriteOptimizationDryRun(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := filepath.Join(td, "test.tpl")
	out := filepath.Join(td, "output.txt")

	// Create template
	if err := os.WriteFile(tpl, []byte("Hello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing output file with same content (with guard to pass guard check)
	if err := os.WriteFile(out, []byte("# #templr generated\nHello World"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Dry-run with unchanged content
	stdout1, _, _ := run(t, bin, "-in", tpl, "-out", out, "-dry-run")

	if !strings.Contains(stdout1, "would skip unchanged") {
		t.Errorf("expected dry-run to mention 'would skip unchanged', got: %s", stdout1)
	}

	// Update template content
	if err := os.WriteFile(tpl, []byte("Hello World Changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Dry-run with changed content
	stdout2, _, _ := run(t, bin, "-in", tpl, "-out", out, "-dry-run")

	if !strings.Contains(stdout2, "(changed)") {
		t.Errorf("expected dry-run to mention '(changed)', got: %s", stdout2)
	}
}

func TestWriteOptimizationWalkMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create templates
	if err := os.WriteFile(filepath.Join(src, "file1.txt.tpl"), []byte("Content 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "file2.txt.tpl"), []byte("Content 2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First walk
	_, _, _ = run(t, bin, "-walk", "-src", src, "-dst", dst)

	// Get mtimes
	info1a, _ := os.Stat(filepath.Join(dst, "file1.txt"))
	info2a, _ := os.Stat(filepath.Join(dst, "file2.txt"))
	mtime1a := info1a.ModTime()
	mtime2a := info2a.ModTime()

	time.Sleep(10 * time.Millisecond)

	// Second walk (no changes)
	_, _, _ = run(t, bin, "-walk", "-src", src, "-dst", dst)

	// Get mtimes again
	info1b, _ := os.Stat(filepath.Join(dst, "file1.txt"))
	info2b, _ := os.Stat(filepath.Join(dst, "file2.txt"))
	mtime1b := info1b.ModTime()
	mtime2b := info2b.ModTime()

	// Both files should have unchanged mtimes
	if !mtime1a.Equal(mtime1b) {
		t.Errorf("file1.txt mtime changed when content was unchanged")
	}
	if !mtime2a.Equal(mtime2b) {
		t.Errorf("file2.txt mtime changed when content was unchanged")
	}

	time.Sleep(10 * time.Millisecond)

	// Modify one template
	if err := os.WriteFile(filepath.Join(src, "file1.txt.tpl"), []byte("Content 1 Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Third walk (file1 changed)
	_, _, _ = run(t, bin, "-walk", "-src", src, "-dst", dst)

	info1c, _ := os.Stat(filepath.Join(dst, "file1.txt"))
	info2c, _ := os.Stat(filepath.Join(dst, "file2.txt"))
	mtime1c := info1c.ModTime()
	mtime2c := info2c.ModTime()

	// file1 should have updated mtime, file2 should not
	if mtime1c.Equal(mtime1b) || mtime1c.Before(mtime1b) {
		t.Errorf("file1.txt mtime should have changed when content was modified")
	}
	if !mtime2c.Equal(mtime2b) {
		t.Errorf("file2.txt mtime should not have changed when content was unchanged")
	}
}

func TestWriteOptimizationWithGuardInjection(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := filepath.Join(td, "test.tpl")
	out := filepath.Join(td, "output.go")

	// Create template (Go file)
	if err := os.WriteFile(tpl, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First render with guard injection (default)
	_, _, _ = run(t, bin, "-in", tpl, "-out", out)

	content1, _ := os.ReadFile(out)
	info1, _ := os.Stat(out)
	mtime1 := info1.ModTime()

	// Verify guard was injected
	if !strings.Contains(string(content1), "#templr generated") {
		t.Errorf("expected guard to be injected in first render")
	}

	time.Sleep(10 * time.Millisecond)

	// Second render with guard injection (content should be identical)
	_, _, _ = run(t, bin, "-in", tpl, "-out", out)

	info2, _ := os.Stat(out)
	mtime2 := info2.ModTime()

	// mtime should NOT change (guard was already there, content unchanged)
	if !mtime1.Equal(mtime2) {
		t.Errorf("expected mtime to remain unchanged when guard is already present")
	}
}

func TestWriteOptimizationDryRunWalkMode(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	src := filepath.Join(td, "src")
	dst := filepath.Join(td, "dst")

	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create templates
	if err := os.WriteFile(filepath.Join(src, "unchanged.txt.tpl"), []byte("Same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "changed.txt.tpl"), []byte("New"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing output for unchanged file (with guard)
	if err := os.WriteFile(filepath.Join(dst, "unchanged.txt"), []byte("# #templr generated\nSame"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing output for changed file (different content, with guard)
	if err := os.WriteFile(filepath.Join(dst, "changed.txt"), []byte("# #templr generated\nOld"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Dry-run walk mode
	stdout, _, _ := run(t, bin, "-walk", "-src", src, "-dst", dst, "-dry-run")

	// Should mention skipping unchanged file
	if !strings.Contains(stdout, "would skip unchanged") || !strings.Contains(stdout, "unchanged.txt") {
		t.Errorf("expected dry-run to mention skipping unchanged.txt, got: %s", stdout)
	}

	// Should mention rendering changed file
	if !strings.Contains(stdout, "(changed)") || !strings.Contains(stdout, "changed.txt") {
		t.Errorf("expected dry-run to mention changed.txt with '(changed)', got: %s", stdout)
	}
}

func TestWriteOptimizationSizeCheckFastPath(t *testing.T) {
	start, _ := os.Getwd()
	bin := buildTemplr(t, start)

	td := t.TempDir()
	tpl := filepath.Join(td, "test.tpl")
	out := filepath.Join(td, "output.txt")

	// Create template with specific content
	if err := os.WriteFile(tpl, []byte("Short"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create existing file with different size (with guard)
	if err := os.WriteFile(out, []byte("# #templr generated\nMuch longer content that is different size"), 0o644); err != nil {
		t.Fatal(err)
	}

	mtime1, _ := os.Stat(out)
	time.Sleep(10 * time.Millisecond)

	// Render (should detect size difference quickly and rewrite)
	_, _, _ = run(t, bin, "-in", tpl, "-out", out)

	// Verify file was rewritten
	content, _ := os.ReadFile(out)
	if !strings.Contains(string(content), "Short") {
		t.Errorf("expected content to contain 'Short', got: %q", string(content))
	}

	mtime2, _ := os.Stat(out)
	if mtime1.ModTime().Equal(mtime2.ModTime()) {
		t.Errorf("expected mtime to change when content size differs")
	}
}
