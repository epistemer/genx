package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/epistemer/genx/internal/manifest"
	"github.com/epistemer/genx/internal/meta"
	"github.com/epistemer/genx/internal/plan"
)

// List writes the steps in the order a run would take them, with what each
// reads and writes (interface.md §9). It runs no producer and no query.
func List(ctx context.Context, opts Options) error {
	s, m, gens, err := load(ctx, opts)
	if err != nil {
		return err
	}
	defer s.root.Close()

	steps := s.steps(m.Producers, gens)
	order, err := plan.Order(steps)
	if err != nil {
		return err
	}
	w := s.opts.Stdout
	for n, i := range order {
		fmt.Fprintf(w, "%d. %s\n", n+1, steps[i].Name)
		if i < len(m.Producers) {
			p := m.Producers[i]
			run := strings.Join(p.Run, " ")
			if p.Dir != "." {
				run += " (in " + p.Dir + ")"
			}
			fmt.Fprintf(w, "   runs    %s\n", run)
			if len(p.Needs) > 0 {
				fmt.Fprintf(w, "   needs   %s\n", joinDomains(p.Needs))
			}
			for _, e := range p.Emits {
				d, ok := s.peekDomain(e)
				switch {
				case e.Foreign:
					fmt.Fprintf(w, "   writes  %s (%s, foreign)\n", e.Path, d)
				case ok:
					fmt.Fprintf(w, "   writes  %s (%s)\n", e.Path, d)
				default:
					fmt.Fprintf(w, "   writes  %s (its domain is known once it exists)\n", e.Path)
				}
			}
			continue
		}
		g := gens[i-len(m.Producers)]
		fmt.Fprintf(w, "   doc     %s\n", g.doc)
		fmt.Fprintf(w, "   needs   %s\n", joinDomains(g.needs))
		if g.template == nil {
			fmt.Fprintf(w, "   emits   %s\n", g.emits)
		}
	}
	return s.listUnswitched(w, m.Use, len(order) > 0)
}

func (s *state) listUnswitched(w io.Writer, uses []manifest.Use, hasSteps bool) error {
	cmd := exec.CommandContext(s.ctx, "go", "list", "-tags=genx", "-e", "-json=ImportPath,Dir,Module,Error", "all")
	cmd.Dir = s.rootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go list all: %v\n%s", err, strings.TrimSpace(stderr.String()))
	}

	type listedPkg struct {
		ImportPath string
		Dir        string
		Module     *struct {
			Path    string
			Version string
		}
		Error *struct {
			Err string
		}
	}

	var pkgs []listedPkg
	seen := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p listedPkg
		if err := dec.Decode(&p); err != nil {
			return fmt.Errorf("go list all: %w", err)
		}
		if !seen[p.ImportPath] {
			seen[p.ImportPath] = true
			pkgs = append(pkgs, p)
		}
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].ImportPath < pkgs[j].ImportPath
	})

	var buf bytes.Buffer
	for _, p := range pkgs {
		if p.Dir == "" {
			continue
		}
		root, err := os.OpenRoot(p.Dir)
		if err != nil {
			continue
		}
		b, err := root.ReadFile(MetaFile)
		root.Close()
		if err != nil {
			continue
		}

		name := p.ImportPath + "/" + MetaFile
		mf, err := meta.Parse(name, b)
		if err != nil {
			firstLine := strings.Split(err.Error(), "\n")[0]
			firstLine = strings.TrimSpace(firstLine)
			firstLine = strings.TrimPrefix(firstLine, name+":")
			firstLine = strings.TrimPrefix(firstLine, MetaFile+":")
			firstLine = strings.TrimSpace(firstLine)
			fmt.Fprintf(&buf, "  warning: %s/%s: %s\n", p.ImportPath, MetaFile, firstLine)
			continue
		}

		modInfo := ""
		if p.Module != nil && p.Module.Version != "" {
			modInfo = fmt.Sprintf(" (%s %s)", p.Module.Path, p.Module.Version)
		}

		for _, g := range mf.Generators {
			if isNamedInUse(uses, p.ImportPath, g.Name) {
				continue
			}
			fmt.Fprintf(&buf, "  %s:%s%s\n      %s\n", p.ImportPath, g.Name, modInfo, g.Doc)
		}
	}

	if buf.Len() > 0 {
		if hasSteps {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Not switched on:")
		if _, err := buf.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}

func isNamedInUse(uses []manifest.Use, pkg, genName string) bool {
	for _, u := range uses {
		if u.Package == pkg {
			for _, g := range u.Generators {
				if g == genName {
					return true
				}
			}
		}
	}
	return false
}
