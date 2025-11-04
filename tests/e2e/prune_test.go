package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kanopi/templr/pkg/templr"
)

// Helper: must make directory in tests
func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

// Helper: must write file in tests
func mustWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", p, err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func TestPruneEmptyDirs_SimpleLeaf(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "foo")
	mustMkdirAll(t, empty)

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got err=%v", empty, err)
	}
	// Root must remain.
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root removed unexpectedly: %v", err)
	}
}

func TestPruneEmptyDirs_NestedCascade(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	mustMkdirAll(t, c)

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}
	// All should be removed since each became empty in cascade.
	for _, p := range []string{c, b, a} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected %s removed, got err=%v", p, err)
		}
	}
}

func TestPruneEmptyDirs_MixedContentStopsCascade(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	mustMkdirAll(t, c)
	// Put a file in a/, so it must not be pruned even if b/c are empty.
	mustWrite(t, filepath.Join(a, "keep.txt"), []byte("x"))

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}
	// c and b should be gone; a should stay due to keep.txt
	if _, err := os.Stat(c); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed", c)
	}
	if _, err := os.Stat(b); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed", b)
	}
	if _, err := os.Stat(a); err != nil {
		t.Fatalf("expected %s to remain: %v", a, err)
	}
}

func TestPruneEmptyDirs_HiddenFilesPreventRemoval(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "dir")
	mustMkdirAll(t, d)
	// A hidden file still makes the dir non-empty
	mustWrite(t, filepath.Join(d, ".keep"), []byte{})

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(d); err != nil {
		t.Fatalf("dir with hidden file should remain: %v", err)
	}
}

func TestPruneEmptyDirs_DoesNotRemoveRoot(t *testing.T) {
	root := t.TempDir()
	// root is empty but should never be removed by the function
	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root should not be removed: %v", err)
	}
}

func TestPruneEmptyDirs_MultipleEmptyBranches(t *testing.T) {
	root := t.TempDir()
	// Create multiple empty branches
	empty1 := filepath.Join(root, "branch1", "subbranch1")
	empty2 := filepath.Join(root, "branch2", "subbranch2")
	mustMkdirAll(t, empty1)
	mustMkdirAll(t, empty2)

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// All empty branches should be removed
	for _, p := range []string{
		empty1,
		filepath.Join(root, "branch1"),
		empty2,
		filepath.Join(root, "branch2"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, got err=%v", p, err)
		}
	}

	// Root should remain
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root should remain: %v", err)
	}
}

func TestPruneEmptyDirs_PartiallyEmptyTree(t *testing.T) {
	root := t.TempDir()
	// Create a tree: root/a/b/c (empty) and root/a/d/file.txt (non-empty)
	emptyBranch := filepath.Join(root, "a", "b", "c")
	fileBranch := filepath.Join(root, "a", "d")
	mustMkdirAll(t, emptyBranch)
	mustWrite(t, filepath.Join(fileBranch, "file.txt"), []byte("content"))

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// Empty branch should be removed
	if _, err := os.Stat(emptyBranch); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed", emptyBranch)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b")); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed", filepath.Join(root, "a", "b"))
	}

	// Branch with file should remain
	if _, err := os.Stat(fileBranch); err != nil {
		t.Fatalf("expected %s to remain: %v", fileBranch, err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); err != nil {
		t.Fatalf("expected %s to remain: %v", filepath.Join(root, "a"), err)
	}
}

func TestPruneEmptyDirs_DeepNesting(t *testing.T) {
	root := t.TempDir()
	// Create a deeply nested empty structure
	deep := root
	for i := 0; i < 10; i++ {
		deep = filepath.Join(deep, "level")
	}
	mustMkdirAll(t, deep)

	if err := templr.PruneEmptyDirs(root); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// All levels should be removed
	check := filepath.Join(root, "level")
	if _, err := os.Stat(check); !os.IsNotExist(err) {
		t.Fatalf("expected deeply nested dirs to be removed, but %s still exists", check)
	}

	// Root should remain
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root should remain: %v", err)
	}
}
