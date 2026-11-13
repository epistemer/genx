package e2e

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoCaseCollisions fails when two files in the repository differ only in
// case. The go command refuses to make a module zip from such a tree, so a
// tag with one can't be fetched (v0.1.0-rc.1 had one).
func TestNoCaseCollisions(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		key := strings.ToLower(rel)
		if other, ok := seen[key]; ok {
			t.Errorf("%s and %s differ only in case", other, rel)
		}
		seen[key] = rel
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
