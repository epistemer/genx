package e2e

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

// TestFailureChecklist keeps docs/failures.md honest: every script a row
// links to exists and contains the row's excerpt (roadmap M9.1).
func TestFailureChecklist(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "failures.md"))
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile(`^\| (.+?) \| (.+) \| \[(\w+)\]\(\.\./internal/e2e/testdata/script/(\w+)\.txtar\) \|$`)
	code := regexp.MustCompile("`([^`]+)`")
	rows := 0
	for n, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Failure ") {
			continue
		}
		m := row.FindStringSubmatch(line)
		if m == nil {
			if !strings.Contains(line, "not scripted") {
				t.Errorf("failures.md:%d: no script link: %s", n+1, line)
			}
			continue
		}
		if m[3] != m[4] {
			t.Errorf("failures.md:%d: link text %s names another script than %s", n+1, m[3], m[4])
		}
		script, err := os.ReadFile(filepath.Join("testdata", "script", m[4]+".txtar"))
		if err != nil {
			t.Errorf("failures.md:%d: %v", n+1, err)
			continue
		}
		// A quoted testscript argument writes ' as ''.
		text := strings.ReplaceAll(string(script), "''", "'")
		excerpts := code.FindAllStringSubmatch(m[2], -1)
		if len(excerpts) == 0 {
			t.Errorf("failures.md:%d: no message excerpt", n+1)
		}
		for _, e := range excerpts {
			if !strings.Contains(text, e[1]) {
				t.Errorf("failures.md:%d: %s.txtar doesn't contain %q", n+1, m[4], e[1])
			}
		}
		rows++
	}
	if rows < 50 {
		t.Errorf("found %d checked rows; the table format may have changed", rows)
	}
}
