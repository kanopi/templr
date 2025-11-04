package templr

import (
	"os"
	"path/filepath"
	"sort"
)

// PruneEmptyDirs removes empty directories under root (bottom-up).
func PruneEmptyDirs(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// deepest-first: longer paths first
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if d == root {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = os.Remove(d)
		}
	}
	return nil
}
