package meta

import (
	"fmt"
	"strings"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/query"
	"github.com/epistemer/genx/internal/yamlx"
	"go.yaml.in/yaml/v4"
)

// Metafile represents a package's generator definitions, genx.meta.yaml.
type Metafile struct {
	Version    int
	Defs       string
	Params     []Param // in the metafile's order
	Generators []Generator
}

// Param is a parameter the metafile declares for all its generators
// (interface amendment A4). Default is a decoded JSON value; nil is null.
type Param struct {
	Name    string
	Doc     string
	Default any
	Line    int
}

// Generator represents a trivial code generator defined in a metafile.
type Generator struct {
	Name      string
	Doc       string
	Needs     []domain.Domain
	Units     string
	Out       string
	Template  string
	Emits     domain.Domain
	HasEmits  bool
	Line      int
	UnitsLine int
	OutLine   int
}

// Parse decodes and validates a metafile from YAML data.
// file is the filename used in error messages.
func Parse(file string, data []byte) (*Metafile, error) {
	root, err := yamlx.Parse(file, data)
	if err != nil {
		return nil, err
	}

	errs := yamlx.NewErrors(file)
	entries := yamlx.MappingEntries(file, root, "", []string{"version", "defs", "params", "generators"}, errs)

	m := &Metafile{}

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

	// defs: optional string
	if defsEntry := yamlx.FindEntry(entries, "defs"); defsEntry != nil {
		defs, ok := yamlx.ReadString(defsEntry.ValNode, "defs", errs)
		if ok {
			m.Defs = defs
		}
	}

	// params: optional mapping of name -> {doc, default}
	if paramsEntry := yamlx.FindEntry(entries, "params"); paramsEntry != nil {
		m.Params = parseParams(file, paramsEntry.ValNode, errs)
	}

	// generators: required, non-empty mapping
	genEntry := yamlx.FindEntry(entries, "generators")
	if genEntry == nil {
		errs.Add(root.Line, "generators", "required")
	} else if genEntry.ValNode.Kind != yaml.MappingNode {
		errs.Add(genEntry.ValNode.Line, "generators", "must be a mapping")
	} else if len(genEntry.ValNode.Content) == 0 {
		errs.Add(genEntry.ValNode.Line, "generators", "must not be empty")
	} else {
		genEntries := yamlx.MappingEntries(file, genEntry.ValNode, "generators", nil, errs)
		for _, ge := range genEntries {
			genName := ge.Key
			genPath := yamlx.FieldPath("generators", genName)

			if !isValidName(genName) {
				errs.Add(ge.KeyNode.Line, genPath, fmt.Sprintf("%q is not a valid generator name (a lowercase letter, then lowercase letters, digits and -)", genName))
			}

			if ge.ValNode.Kind != yaml.MappingNode {
				errs.Add(ge.ValNode.Line, genPath, "must be a mapping")
				continue
			}

			g := parseGenerator(file, genName, ge.KeyNode.Line, ge.ValNode, genPath, errs)
			m.Generators = append(m.Generators, g)
		}
	}

	if err := errs.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func parseGenerator(file, name string, keyLine int, node *yaml.Node, path string, errs *yamlx.Errors) Generator {
	g := Generator{
		Name: name,
		Line: keyLine,
	}

	fields := yamlx.MappingEntries(file, node, path, []string{"doc", "needs", "units", "out", "template", "emits"}, errs)

	// doc: required, one line
	docEntry := yamlx.FindEntry(fields, "doc")
	docPath := yamlx.FieldPath(path, "doc")
	if docEntry == nil {
		errs.Add(node.Line, docPath, "required")
	} else {
		doc, ok := yamlx.ReadString(docEntry.ValNode, docPath, errs)
		if ok {
			if doc == "" {
				errs.Add(docEntry.ValNode.Line, docPath, "must not be empty")
			} else if strings.ContainsAny(doc, "\r\n") {
				errs.Add(docEntry.ValNode.Line, docPath, "must be one line")
			} else {
				g.Doc = doc
			}
		}
	}

	// needs: required, non-empty sequence of domains
	needsEntry := yamlx.FindEntry(fields, "needs")
	needsPath := yamlx.FieldPath(path, "needs")
	if needsEntry == nil {
		errs.Add(node.Line, needsPath, "required")
	} else if needsEntry.ValNode.Kind != yaml.SequenceNode {
		errs.Add(needsEntry.ValNode.Line, needsPath, "must be a sequence")
	} else if len(needsEntry.ValNode.Content) == 0 {
		errs.Add(needsEntry.ValNode.Line, needsPath, "must not be empty")
	} else {
		for j, child := range needsEntry.ValNode.Content {
			elemPath := yamlx.Index(needsPath, j)
			dStr, ok := yamlx.ReadString(child, elemPath, errs)
			if ok {
				dom, err := domain.Parse(dStr)
				if err != nil {
					errs.Add(child.Line, elemPath, err.Error())
				} else {
					g.Needs = append(g.Needs, dom)
				}
			}
		}
	}

	// units: required, non-empty
	unitsEntry := yamlx.FindEntry(fields, "units")
	unitsPath := yamlx.FieldPath(path, "units")
	if unitsEntry == nil {
		errs.Add(node.Line, unitsPath, "required")
	} else {
		g.UnitsLine = unitsEntry.ValNode.Line
		units, ok := yamlx.ReadString(unitsEntry.ValNode, unitsPath, errs)
		if ok {
			if units == "" {
				errs.Add(unitsEntry.ValNode.Line, unitsPath, "must not be empty")
			} else {
				g.Units = units
			}
		}
	}

	// out: required, non-empty
	outEntry := yamlx.FindEntry(fields, "out")
	outPath := yamlx.FieldPath(path, "out")
	if outEntry == nil {
		errs.Add(node.Line, outPath, "required")
	} else {
		g.OutLine = outEntry.ValNode.Line
		out, ok := yamlx.ReadString(outEntry.ValNode, outPath, errs)
		if ok {
			if out == "" {
				errs.Add(outEntry.ValNode.Line, outPath, "must not be empty")
			} else {
				g.Out = out
			}
		}
	}

	// exactly one of template / emits
	tmplEntry := yamlx.FindEntry(fields, "template")
	emitsEntry := yamlx.FindEntry(fields, "emits")

	if (tmplEntry == nil && emitsEntry == nil) || (tmplEntry != nil && emitsEntry != nil) {
		errs.Add(keyLine, path, "exactly one of template and emits")
	}

	if tmplEntry != nil {
		tmplPath := yamlx.FieldPath(path, "template")
		tmpl, ok := yamlx.ReadString(tmplEntry.ValNode, tmplPath, errs)
		if ok {
			if tmpl == "" {
				errs.Add(tmplEntry.ValNode.Line, tmplPath, "must not be empty")
			} else if strings.Contains(tmpl, "/") || strings.Contains(tmpl, "\\") {
				errs.Add(tmplEntry.ValNode.Line, tmplPath, "must not contain '/' or '\\'")
			} else if tmpl == "." || tmpl == ".." {
				errs.Add(tmplEntry.ValNode.Line, tmplPath, "must not be '.' or '..'")
			} else if strings.HasPrefix(tmpl, ".") {
				errs.Add(tmplEntry.ValNode.Line, tmplPath, "must not start with '.'")
			} else {
				g.Template = tmpl
			}
		}
	}

	if emitsEntry != nil {
		emitsPath := yamlx.FieldPath(path, "emits")
		eStr, ok := yamlx.ReadString(emitsEntry.ValNode, emitsPath, errs)
		if ok {
			dom, err := domain.Parse(eStr)
			if err != nil {
				errs.Add(emitsEntry.ValNode.Line, emitsPath, err.Error())
			} else {
				g.Emits = dom
				g.HasEmits = true
			}
		}
	}

	return g
}

func parseParams(file string, node *yaml.Node, errs *yamlx.Errors) []Param {
	if node.Kind != yaml.MappingNode {
		errs.Add(node.Line, "params", "must be a mapping of parameter names to {doc, default}")
		return nil
	}
	var params []Param
	for _, pe := range yamlx.MappingEntries(file, node, "params", nil, errs) {
		path := yamlx.FieldPath("params", pe.Key)
		if !isValidParamName(pe.Key) {
			errs.Add(pe.KeyNode.Line, path, fmt.Sprintf("%q is not a valid parameter name (a lowercase letter, then lowercase letters, digits and _)", pe.Key))
			continue
		}
		if pe.ValNode.Kind != yaml.MappingNode {
			errs.Add(pe.ValNode.Line, path, "must be a mapping {doc, default}")
			continue
		}
		p := Param{Name: pe.Key, Line: pe.KeyNode.Line}
		fields := yamlx.MappingEntries(file, pe.ValNode, path, []string{"doc", "default"}, errs)
		docPath := yamlx.FieldPath(path, "doc")
		if docEntry := yamlx.FindEntry(fields, "doc"); docEntry == nil {
			errs.Add(pe.KeyNode.Line, docPath, "required")
		} else if doc, ok := yamlx.ReadString(docEntry.ValNode, docPath, errs); ok {
			if doc == "" {
				errs.Add(docEntry.ValNode.Line, docPath, "must not be empty")
			} else if strings.ContainsAny(doc, "\r\n") {
				errs.Add(docEntry.ValNode.Line, docPath, "must be one line")
			} else {
				p.Doc = doc
			}
		}
		defPath := yamlx.FieldPath(path, "default")
		if defEntry := yamlx.FindEntry(fields, "default"); defEntry == nil {
			errs.Add(pe.KeyNode.Line, defPath, "required (write default: null for a parameter with no default value)")
		} else if v, err := query.DecodeNode(defEntry.ValNode); err != nil {
			errs.Add(defEntry.ValNode.Line, defPath, err.Error())
		} else {
			p.Default = v
		}
		params = append(params, p)
	}
	return params
}

// isValidParamName reports whether s matches [a-z][a-z0-9_]*, so a query
// reads it as $params.<name>.
func isValidParamName(s string) bool {
	if len(s) == 0 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
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
