package run

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/manifest"
	"github.com/epistemer/genx/internal/meta"
	"github.com/epistemer/genx/internal/query"
	"github.com/epistemer/genx/internal/render"
)

// MetaFile is a metafile's name in its package's directory.
const MetaFile = "genx.meta.yaml"

// generator is a switched-on generator, compiled and ready to run.
type generator struct {
	step     string // "<package>:<name>"
	needs    []domain.Domain
	units    *query.Query
	out      *query.Query
	template *render.Template // nil when the generator emits a domain
	emits    domain.Domain
}

// resolve finds the directory of every package in use with one go list
// call, reads each metafile, and compiles the generators the manifest
// switches on, in use order.
func (s *state) resolve(uses []manifest.Use) ([]generator, error) {
	if len(uses) == 0 {
		return nil, nil
	}
	dirs, err := s.packageDirs(uses)
	if err != nil {
		return nil, err
	}
	var gens []generator
	for _, u := range uses {
		g, err := compileUse(u, dirs[u.Package])
		if err != nil {
			return nil, err
		}
		gens = append(gens, g...)
	}
	return gens, nil
}

// compileUse compiles the generators one use entry switches on.
func compileUse(u manifest.Use, dir string) ([]generator, error) {
	mf, root, err := readMetafile(u.Package, dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var gens []generator
	for _, name := range u.Generators {
		g, err := compile(u.Package, mf, root, name)
		if err != nil {
			return nil, err
		}
		gens = append(gens, g)
	}
	return gens, nil
}

// packageDirs lists the packages' directories with the go command, which
// applies the module's replace directives, go.work and -mod=vendor.
func (s *state) packageDirs(uses []manifest.Use) (map[string]string, error) {
	args := []string{"list", "-e", "-f", "{{.ImportPath}}\t{{.Dir}}\t{{if .Error}}{{.Error.Err}}{{end}}"}
	seen := map[string]bool{}
	for _, u := range uses {
		if !seen[u.Package] {
			seen[u.Package] = true
			args = append(args, u.Package)
		}
	}
	cmd := exec.CommandContext(s.ctx, "go", args...)
	cmd.Dir = s.rootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %v\n%s", err, strings.TrimSpace(stderr.String()))
	}
	dirs := map[string]string{}
	for line := range strings.Lines(string(out)) {
		f := strings.SplitN(strings.TrimRight(line, "\r\n"), "\t", 3)
		if len(f) != 3 {
			continue
		}
		if f[2] != "" || f[1] == "" {
			return nil, fmt.Errorf("use %s: the go command can't find this package: %s\n"+
				"If nothing in the app imports it yet, go mod tidy removed it from go.mod, and go mod vendor didn't copy it.\n"+
				"Keep a file in the app with a blank import, which the build leaves out:\n\n"+
				"\t//go:build genx\n\n\tpackage <name>\n\n\timport _ %q\n\n"+
				"then run go get %s", f[0], strings.TrimSpace(f[2]), f[0], f[0])
		}
		dirs[f[0]] = f[1]
	}
	for p := range seen {
		if dirs[p] == "" {
			return nil, fmt.Errorf("use %s: go list didn't report this package", p)
		}
	}
	return dirs, nil
}

// readMetafile opens the package's directory as a root and parses its
// metafile. The caller closes the root.
func readMetafile(pkg, dir string) (*meta.Metafile, *os.Root, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("use %s: %w", pkg, err)
	}
	name := pkg + "/" + MetaFile
	b, err := root.ReadFile(MetaFile)
	if errors.Is(err, os.ErrNotExist) {
		root.Close()
		return nil, nil, fmt.Errorf("use %s: the package has no %s", pkg, MetaFile)
	}
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	mf, err := meta.Parse(name, b)
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	return mf, root, nil
}

// compile prepares one generator of a metafile.
func compile(pkg string, mf *meta.Metafile, root *os.Root, name string) (generator, error) {
	var mg *meta.Generator
	for i := range mf.Generators {
		if mf.Generators[i].Name == name {
			mg = &mf.Generators[i]
		}
	}
	if mg == nil {
		var names []string
		for _, g := range mf.Generators {
			names = append(names, g.Name)
		}
		return generator{}, fmt.Errorf("use %s: no generator %q in its %s (it has: %s)", pkg, name, MetaFile, strings.Join(names, ", "))
	}
	step := pkg + ":" + name
	g := generator{step: step, needs: mg.Needs, emits: mg.Emits}
	var err error
	if g.units, err = query.Compile(fmt.Sprintf("%s/%s:%d: units", pkg, MetaFile, mg.UnitsLine), mf.Defs, mg.Units); err != nil {
		return generator{}, err
	}
	if g.out, err = query.Compile(fmt.Sprintf("%s/%s:%d: out", pkg, MetaFile, mg.OutLine), mf.Defs, mg.Out); err != nil {
		return generator{}, err
	}
	if mg.HasEmits {
		return g, nil
	}
	src, err := root.ReadFile(mg.Template)
	if err != nil {
		return generator{}, fmt.Errorf("%s: template: %w", step, err)
	}
	if g.template, err = render.Parse(mg.Template, src); err != nil {
		return generator{}, fmt.Errorf("%s: %w", step, err)
	}
	return g, nil
}
