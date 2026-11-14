package manifest

import (
	"fmt"
	"strings"
	"time"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/paths"
	"github.com/epistemer/genx/internal/query"
	"github.com/epistemer/genx/internal/yamlx"
	"go.yaml.in/yaml/v4"
)

// Manifest represents an app's generation manifest, genx.yaml.
type Manifest struct {
	Version   int
	Producers []Producer
	Use       []Use
}

// Producer represents a command that writes domain files.
type Producer struct {
	Name    string
	Dir     string
	Run     []string
	Check   []string
	Timeout time.Duration
	Needs   []domain.Domain
	Emits   []Emit
	Line    int
}

// Emit represents a domain file written by a producer.
type Emit struct {
	Path    string
	Domain  domain.Domain
	Foreign bool
	Line    int
}

// Use represents a package whose generators are activated.
type Use struct {
	Package    string
	Generators []string
	// Params are the values the entry's map form sets, in the order
	// written (interface amendment A4). The list form sets none.
	Params []Param
	// Path is the entry's field path in messages, such as
	// "use[0].example.com/app/gen".
	Path string
	Line int
}

// Param is a value an app gives one of a package's parameters. Value is a
// decoded JSON value; nil is null.
type Param struct {
	Name  string
	Value any
	Line  int
}

// Parse decodes and validates a manifest from YAML data.
// file is the filename used in error messages.
func Parse(file string, data []byte) (*Manifest, error) {
	root, err := yamlx.Parse(file, data)
	if err != nil {
		return nil, err
	}

	errs := yamlx.NewErrors(file)
	entries := yamlx.MappingEntries(file, root, "", []string{"version", "producers", "use"}, errs)

	m := &Manifest{}

	// version: required and == 1
	verEntry := yamlx.FindEntry(entries, "version")
	if verEntry == nil {
		errs.Add(root.Line, "version", "required")
	} else {
		v, ok := yamlx.ReadInt(verEntry.ValNode, "version", errs)
		if ok {
			if v > 1 {
				errs.Add(verEntry.ValNode.Line, "version", fmt.Sprintf("version %d is newer than this genx supports; upgrade genx", v))
			} else if v < 1 {
				errs.Add(verEntry.ValNode.Line, "version", "must be 1")
			} else {
				m.Version = v
			}
		}
	}

	seenProducers := make(map[string]bool)
	seenEmits := make(map[string]int) // folded path -> line

	// producers: optional list
	if prodsEntry := yamlx.FindEntry(entries, "producers"); prodsEntry != nil {
		if prodsEntry.ValNode.Kind != yaml.SequenceNode {
			errs.Add(prodsEntry.ValNode.Line, "producers", "must be a sequence")
		} else {
			for i, pNode := range prodsEntry.ValNode.Content {
				pPath := yamlx.Index("producers", i)
				p := parseProducer(file, pNode, pPath, seenProducers, seenEmits, errs)
				m.Producers = append(m.Producers, p)
			}
		}
	}

	seenUse := make(map[string]bool) // "pkg:gen" -> true

	// use: optional list of one-key mappings
	if useEntry := yamlx.FindEntry(entries, "use"); useEntry != nil {
		if useEntry.ValNode.Kind != yaml.SequenceNode {
			errs.Add(useEntry.ValNode.Line, "use", "must be a sequence")
		} else {
			for k, uNode := range useEntry.ValNode.Content {
				uPath := yamlx.Index("use", k)
				u, ok := parseUse(file, uNode, uPath, seenUse, errs)
				if ok {
					m.Use = append(m.Use, u)
				}
			}
		}
	}

	if err := errs.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func parseProducer(file string, node *yaml.Node, path string, seenProducers map[string]bool, seenEmits map[string]int, errs *yamlx.Errors) Producer {
	p := Producer{
		Dir:     ".",
		Timeout: 10 * time.Minute,
		Line:    node.Line,
	}

	entries := yamlx.MappingEntries(file, node, path, []string{"name", "dir", "run", "check", "timeout", "needs", "emits"}, errs)

	// name: required, pattern [a-z][a-z0-9-]*, unique
	nameEntry := yamlx.FindEntry(entries, "name")
	namePath := yamlx.FieldPath(path, "name")
	if nameEntry == nil {
		errs.Add(node.Line, namePath, "required")
	} else {
		name, ok := yamlx.ReadString(nameEntry.ValNode, namePath, errs)
		if ok {
			if !isValidName(name) {
				errs.Add(nameEntry.ValNode.Line, namePath, fmt.Sprintf("%q is not a valid producer name (a lowercase letter, then lowercase letters, digits and -)", name))
			} else if seenProducers[name] {
				errs.Add(nameEntry.ValNode.Line, namePath, fmt.Sprintf("duplicate producer name %q", name))
			} else {
				seenProducers[name] = true
				p.Name = name
			}
		}
	}

	// dir: optional, default ".", paths.Dir
	if dirEntry := yamlx.FindEntry(entries, "dir"); dirEntry != nil {
		dirPath := yamlx.FieldPath(path, "dir")
		d, ok := yamlx.ReadString(dirEntry.ValNode, dirPath, errs)
		if ok {
			if err := paths.Dir(d); err != nil {
				errs.Add(dirEntry.ValNode.Line, dirPath, err.Error())
			} else {
				p.Dir = d
			}
		}
	}

	// run: required, non-empty argv of non-empty strings
	runEntry := yamlx.FindEntry(entries, "run")
	runPath := yamlx.FieldPath(path, "run")
	if runEntry == nil {
		errs.Add(node.Line, runPath, "required")
	} else {
		p.Run = parseArgv(runEntry.ValNode, runPath, errs)
	}

	// check: optional, non-empty argv of non-empty strings
	if checkEntry := yamlx.FindEntry(entries, "check"); checkEntry != nil {
		checkPath := yamlx.FieldPath(path, "check")
		p.Check = parseArgv(checkEntry.ValNode, checkPath, errs)
	}

	// timeout: optional, positive Go duration, default 10m
	if timeoutEntry := yamlx.FindEntry(entries, "timeout"); timeoutEntry != nil {
		timeoutPath := yamlx.FieldPath(path, "timeout")
		tStr, ok := yamlx.ReadString(timeoutEntry.ValNode, timeoutPath, errs)
		if ok {
			dur, err := time.ParseDuration(tStr)
			if err != nil {
				errs.Add(timeoutEntry.ValNode.Line, timeoutPath, fmt.Sprintf("invalid duration %q", tStr))
			} else if dur <= 0 {
				errs.Add(timeoutEntry.ValNode.Line, timeoutPath, "must be positive")
			} else {
				p.Timeout = dur
			}
		}
	}

	// needs: optional, sequence of domains
	if needsEntry := yamlx.FindEntry(entries, "needs"); needsEntry != nil {
		needsPath := yamlx.FieldPath(path, "needs")
		if needsEntry.ValNode.Kind != yaml.SequenceNode {
			errs.Add(needsEntry.ValNode.Line, needsPath, "must be a sequence")
		} else {
			for j, child := range needsEntry.ValNode.Content {
				elemPath := yamlx.Index(needsPath, j)
				dStr, ok := yamlx.ReadString(child, elemPath, errs)
				if ok {
					dom, err := domain.Parse(dStr)
					if err != nil {
						errs.Add(child.Line, elemPath, err.Error())
					} else {
						p.Needs = append(p.Needs, dom)
					}
				}
			}
		}
	}

	// emits: required, non-empty sequence
	emitsEntry := yamlx.FindEntry(entries, "emits")
	emitsPath := yamlx.FieldPath(path, "emits")
	if emitsEntry == nil {
		errs.Add(node.Line, emitsPath, "required")
	} else if emitsEntry.ValNode.Kind != yaml.SequenceNode {
		errs.Add(emitsEntry.ValNode.Line, emitsPath, "must be a sequence")
	} else if len(emitsEntry.ValNode.Content) == 0 {
		errs.Add(emitsEntry.ValNode.Line, emitsPath, "must not be empty")
	} else {
		for j, child := range emitsEntry.ValNode.Content {
			elemPath := yamlx.Index(emitsPath, j)
			emit, ok := parseEmit(file, child, elemPath, seenEmits, errs)
			if ok {
				p.Emits = append(p.Emits, emit)
			}
		}
	}

	return p
}

func parseArgv(node *yaml.Node, path string, errs *yamlx.Errors) []string {
	if node.Kind != yaml.SequenceNode {
		errs.Add(node.Line, path, "must be a sequence of strings")
		return nil
	}
	if len(node.Content) == 0 {
		errs.Add(node.Line, path, "must not be empty")
		return nil
	}
	var argv []string
	for j, child := range node.Content {
		elemPath := yamlx.Index(path, j)
		if child.Kind != yaml.ScalarNode || (child.Tag != "!!str" && child.Tag != "") {
			errs.Add(child.Line, elemPath, "must be a string")
			continue
		}
		if child.Value == "" {
			errs.Add(child.Line, elemPath, "must not be empty")
			continue
		}
		argv = append(argv, child.Value)
	}
	return argv
}

func parseEmit(file string, node *yaml.Node, path string, seenEmits map[string]int, errs *yamlx.Errors) (Emit, bool) {
	emit := Emit{Line: node.Line}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag != "!!str" && node.Tag != "" {
			errs.Add(node.Line, path, "must be a string or a mapping {path, domain}")
			return emit, false
		}
		p := node.Value
		if err := paths.Declared(p); err != nil {
			errs.Add(node.Line, path, err.Error())
		} else {
			checkDuplicateEmit(node.Line, path, p, seenEmits, errs)
		}
		emit.Path = p
		emit.Foreign = false
		return emit, true

	case yaml.MappingNode:
		entries := yamlx.MappingEntries(file, node, path, []string{"path", "domain"}, errs)
		pathEntry := yamlx.FindEntry(entries, "path")
		domEntry := yamlx.FindEntry(entries, "domain")

		pPath := yamlx.FieldPath(path, "path")
		dPath := yamlx.FieldPath(path, "domain")

		valid := true
		if pathEntry == nil {
			errs.Add(node.Line, pPath, "required")
			valid = false
		} else {
			p, ok := yamlx.ReadString(pathEntry.ValNode, pPath, errs)
			if ok {
				if err := paths.Declared(p); err != nil {
					errs.Add(pathEntry.ValNode.Line, pPath, err.Error())
				} else {
					checkDuplicateEmit(pathEntry.ValNode.Line, pPath, p, seenEmits, errs)
				}
				emit.Path = p
			} else {
				valid = false
			}
		}

		if domEntry == nil {
			errs.Add(node.Line, dPath, "required")
			valid = false
		} else {
			dStr, ok := yamlx.ReadString(domEntry.ValNode, dPath, errs)
			if ok {
				dom, err := domain.Parse(dStr)
				if err != nil {
					errs.Add(domEntry.ValNode.Line, dPath, err.Error())
				} else {
					emit.Domain = dom
				}
			} else {
				valid = false
			}
		}

		emit.Foreign = true
		return emit, valid

	default:
		errs.Add(node.Line, path, "must be a string or a mapping {path, domain}")
		return emit, false
	}
}

func checkDuplicateEmit(line int, path, p string, seenEmits map[string]int, errs *yamlx.Errors) {
	folded := paths.Fold(p)
	if first, exists := seenEmits[folded]; exists {
		errs.Add(line, path, fmt.Sprintf("duplicate emit path %q (also line %d)", p, first))
		return
	}
	seenEmits[folded] = line
}

func parseUse(file string, node *yaml.Node, path string, seenUse map[string]bool, errs *yamlx.Errors) (Use, bool) {
	u := Use{Line: node.Line}
	if node.Kind != yaml.MappingNode {
		errs.Add(node.Line, path, "must be a one-key mapping")
		return u, false
	}

	entries := yamlx.MappingEntries(file, node, path, nil, errs)
	if len(entries) != 1 {
		errs.Add(node.Line, path, "must be a one-key mapping")
		return u, false
	}

	entry := entries[0]
	pkg := entry.Key
	pkgNode := entry.KeyNode
	valNode := entry.ValNode

	if pkg == "" {
		errs.Add(pkgNode.Line, path, "package path must not be empty")
	} else if strings.ContainsAny(pkg, " \t\r\n") {
		errs.Add(pkgNode.Line, path, "package path must not contain spaces")
	}

	pkgFieldPath := yamlx.FieldPath(path, pkg)
	u.Package = pkg
	u.Path = pkgFieldPath
	gensNode, gensPath := valNode, pkgFieldPath
	switch valNode.Kind {
	case yaml.SequenceNode:
	case yaml.MappingNode:
		// The map form: {generators: [...], params: {name: value}}.
		fields := yamlx.MappingEntries(file, valNode, pkgFieldPath, []string{"generators", "params"}, errs)
		if paramsEntry := yamlx.FindEntry(fields, "params"); paramsEntry != nil {
			u.Params = parseParams(file, paramsEntry.ValNode, yamlx.FieldPath(pkgFieldPath, "params"), errs)
		}
		gensPath = yamlx.FieldPath(pkgFieldPath, "generators")
		gensEntry := yamlx.FindEntry(fields, "generators")
		if gensEntry == nil {
			errs.Add(pkgNode.Line, gensPath, "required")
			return u, false
		}
		gensNode = gensEntry.ValNode
		if gensNode.Kind != yaml.SequenceNode {
			errs.Add(gensNode.Line, gensPath, "must be a sequence of generator names")
			return u, false
		}
	default:
		errs.Add(valNode.Line, pkgFieldPath, "must be a sequence of generator names, or a mapping {generators, params}")
		return u, false
	}
	if len(gensNode.Content) == 0 {
		errs.Add(gensNode.Line, gensPath, "must not be empty")
		return u, false
	}

	for _, genNode := range gensNode.Content {
		if genNode.Kind != yaml.ScalarNode || (genNode.Tag != "!!str" && genNode.Tag != "") {
			errs.Add(genNode.Line, gensPath, "generator name must be a string")
			continue
		}
		gen := genNode.Value
		if !isValidName(gen) {
			errs.Add(genNode.Line, gensPath, fmt.Sprintf("%q is not a valid generator name (a lowercase letter, then lowercase letters, digits and -)", gen))
			continue
		}
		pairKey := pkg + ":" + gen
		if seenUse[pairKey] {
			errs.Add(genNode.Line, gensPath, fmt.Sprintf("duplicate generator %q for package %q", gen, pkg))
			continue
		}
		seenUse[pairKey] = true
		u.Generators = append(u.Generators, gen)
	}

	return u, true
}

// parseParams reads the values a use entry sets. Whether the package
// declares each one, and its type, are checked once its metafile is read.
func parseParams(file string, node *yaml.Node, path string, errs *yamlx.Errors) []Param {
	if node.Kind != yaml.MappingNode {
		errs.Add(node.Line, path, "must be a mapping of parameter names to values")
		return nil
	}
	var params []Param
	for _, e := range yamlx.MappingEntries(file, node, path, nil, errs) {
		v, err := query.DecodeNode(e.ValNode)
		if err != nil {
			errs.Add(query.ErrorLine(err, e.ValNode.Line), yamlx.FieldPath(path, e.Key), err.Error())
			continue
		}
		params = append(params, Param{Name: e.Key, Value: v, Line: e.KeyNode.Line})
	}
	return params
}

func isValidName(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
