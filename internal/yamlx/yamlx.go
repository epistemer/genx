// Package yamlx provides strict YAML helpers for parsing and validating
// manifests and metafiles with line numbers, field paths, and ordered error reporting.
package yamlx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Error represents a located error in a YAML document.
type Error struct {
	File string
	Line int
	Path string
	Msg  string
}

func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Msg)
	}
	return fmt.Sprintf("%s:%d: %s: %s", e.File, e.Line, e.Path, e.Msg)
}

// Errors collects errors for a file and produces a joined error in file order.
type Errors struct {
	file string
	errs []*Error
}

// NewErrors creates an Errors accumulator for the given file.
func NewErrors(file string) *Errors {
	return &Errors{file: file}
}

// Add appends an error at the given line and field path.
func (e *Errors) Add(line int, path, msg string) {
	if line <= 0 {
		line = 1
	}
	e.errs = append(e.errs, &Error{
		File: e.file,
		Line: line,
		Path: path,
		Msg:  msg,
	})
}

// AddErr appends an existing error message at the given line and field path.
func (e *Errors) AddErr(line int, path string, err error) {
	if err != nil {
		e.Add(line, path, err.Error())
	}
}

// Err returns all collected errors joined with errors.Join, sorted in file order.
func (e *Errors) Err() error {
	if len(e.errs) == 0 {
		return nil
	}
	sort.SliceStable(e.errs, func(i, j int) bool {
		if e.errs[i].Line != e.errs[j].Line {
			return e.errs[i].Line < e.errs[j].Line
		}
		return e.errs[i].Path < e.errs[j].Path
	})
	list := make([]error, len(e.errs))
	for i, err := range e.errs {
		list[i] = err
	}
	return errors.Join(list...)
}

// FieldPath joins a base path and a child field key using "a.b" syntax.
func FieldPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// Index returns a field path with an array index, e.g. "a[2]".
func Index(base string, i int) string {
	return fmt.Sprintf("%s[%d]", base, i)
}

// Entry is an ordered key-value pair from a mapping node.
type Entry struct {
	Key     string
	KeyNode *yaml.Node
	ValNode *yaml.Node
}

// Parse decodes exactly one YAML document into a root *yaml.Node.
// An empty document, syntax error, a second document, non-scalar mapping keys,
// or duplicate mapping keys return errors. Aliases are expanded.
func Parse(file string, data []byte) (*yaml.Node, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, &Error{File: file, Line: 1, Msg: "empty document"}
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, &Error{File: file, Line: 1, Msg: "empty document"}
		}
		line := extractLine(err)
		msg := cleanYAMLError(err)
		return nil, &Error{File: file, Line: line, Msg: msg}
	}

	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 || (len(doc.Content) == 1 && doc.Content[0].Tag == "!!null") {
			return nil, &Error{File: file, Line: 1, Msg: "empty document"}
		}
	}

	var doc2 yaml.Node
	if err := dec.Decode(&doc2); err == nil {
		line := doc2.Line
		if line <= 0 {
			line = 1
		}
		return nil, &Error{File: file, Line: line, Msg: "unexpected second document"}
	}

	root := doc.Content[0]

	visited := make(map[*yaml.Node]bool)
	if err := expandAliases(file, root, "", visited); err != nil {
		return nil, err
	}

	errs := NewErrors(file)
	validateTree(root, "", errs)
	if err := errs.Err(); err != nil {
		return nil, err
	}

	return root, nil
}

func extractLine(err error) int {
	s := err.Error()
	const prefix = "yaml: line "
	if strings.HasPrefix(s, prefix) {
		rest := s[len(prefix):]
		if idx := strings.IndexByte(rest, ':'); idx != -1 {
			if line, parseErr := strconv.Atoi(rest[:idx]); parseErr == nil {
				return line
			}
		}
	}
	return 1
}

func cleanYAMLError(err error) string {
	s := err.Error()
	s = strings.TrimPrefix(s, "yaml: ")
	if idx := strings.Index(s, ": "); idx != -1 && strings.HasPrefix(s, "line ") {
		s = s[idx+2:]
	}
	return s
}

func cloneNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	cp := *n
	if n.Content != nil {
		cp.Content = make([]*yaml.Node, len(n.Content))
		for i, c := range n.Content {
			cp.Content[i] = cloneNode(c)
		}
	}
	return &cp
}

func expandAliases(file string, n *yaml.Node, path string, visited map[*yaml.Node]bool) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		if visited[n] {
			return &Error{File: file, Line: n.Line, Path: path, Msg: "alias cycle detected"}
		}
		if n.Alias == nil {
			return &Error{File: file, Line: n.Line, Path: path, Msg: "unresolved alias"}
		}
		visited[n] = true
		if err := expandAliases(file, n.Alias, path, visited); err != nil {
			return err
		}
		line := n.Line
		col := n.Column
		target := cloneNode(n.Alias)
		*n = *target
		if line > 0 {
			n.Line = line
			n.Column = col
		}
		visited[n] = false
	}
	for _, child := range n.Content {
		if err := expandAliases(file, child, path, visited); err != nil {
			return err
		}
	}
	return nil
}

func validateTree(n *yaml.Node, path string, errs *Errors) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.MappingNode:
		seen := make(map[string]bool)
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			v := n.Content[i+1]
			if k.Kind != yaml.ScalarNode {
				errs.Add(k.Line, path, "mapping key must be scalar")
				continue
			}
			keyPath := FieldPath(path, k.Value)
			if seen[k.Value] {
				errs.Add(k.Line, keyPath, fmt.Sprintf("duplicate key %q", k.Value))
			}
			seen[k.Value] = true
			validateTree(v, keyPath, errs)
		}
	case yaml.SequenceNode:
		for i, elem := range n.Content {
			elemPath := Index(path, i)
			validateTree(elem, elemPath, errs)
		}
	}
}

// MappingEntries reads a mapping node into ordered key/value entries,
// verifying that keys are scalars, non-duplicate, and (if allowedKeys is non-nil)
// contained in the allowedKeys list.
func MappingEntries(file string, node *yaml.Node, path string, allowedKeys []string, errs *Errors) []Entry {
	if node == nil || node.Kind != yaml.MappingNode {
		line := 1
		if node != nil && node.Line > 0 {
			line = node.Line
		}
		errs.Add(line, path, "must be a mapping")
		return nil
	}

	var allowedMap map[string]bool
	if allowedKeys != nil {
		allowedMap = make(map[string]bool, len(allowedKeys))
		for _, k := range allowedKeys {
			allowedMap[k] = true
		}
	}

	var entries []Entry
	seen := make(map[string]bool)

	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		v := node.Content[i+1]

		if k.Kind != yaml.ScalarNode {
			errs.Add(k.Line, path, "mapping key must be scalar")
			continue
		}

		keyPath := FieldPath(path, k.Value)
		if seen[k.Value] {
			errs.Add(k.Line, keyPath, fmt.Sprintf("duplicate key %q", k.Value))
		}
		seen[k.Value] = true

		if allowedMap != nil && !allowedMap[k.Value] {
			errs.Add(k.Line, keyPath, "unknown field")
			continue
		}

		entries = append(entries, Entry{
			Key:     k.Value,
			KeyNode: k,
			ValNode: v,
		})
	}
	return entries
}

// FindEntry returns the entry matching key, or nil if not present.
func FindEntry(entries []Entry, key string) *Entry {
	for i := range entries {
		if entries[i].Key == key {
			return &entries[i]
		}
	}
	return nil
}

// ReadString reads a scalar string from node.
// Returns "", false if node is nil or not a scalar string.
func ReadString(node *yaml.Node, path string, errs *Errors) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Kind != yaml.ScalarNode || (node.Tag != "!!str" && node.Tag != "") {
		errs.Add(node.Line, path, "must be a string")
		return "", false
	}
	return node.Value, true
}

// ReadInt reads a scalar integer from node.
// Returns 0, false if node is nil or not an integer.
func ReadInt(node *yaml.Node, path string, errs *Errors) (int, bool) {
	if node == nil {
		return 0, false
	}
	if node.Kind != yaml.ScalarNode || node.Tag == "!!str" {
		errs.Add(node.Line, path, "must be an integer")
		return 0, false
	}
	val, err := strconv.Atoi(node.Value)
	if err != nil {
		errs.Add(node.Line, path, "must be an integer")
		return 0, false
	}
	return val, true
}

// ReadStringSlice reads a sequence of scalar strings from node.
// Returns nil, false if node is nil or not a sequence of strings.
func ReadStringSlice(node *yaml.Node, path string, errs *Errors) ([]string, bool) {
	if node == nil {
		return nil, false
	}
	if node.Kind != yaml.SequenceNode {
		errs.Add(node.Line, path, "must be a sequence")
		return nil, false
	}
	var res []string
	ok := true
	for i, child := range node.Content {
		elemPath := Index(path, i)
		if child.Kind != yaml.ScalarNode || (child.Tag != "!!str" && child.Tag != "") {
			errs.Add(child.Line, elemPath, "must be a string")
			ok = false
			continue
		}
		res = append(res, child.Value)
	}
	if !ok {
		return nil, false
	}
	return res, true
}
