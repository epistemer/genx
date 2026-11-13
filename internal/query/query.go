package query

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"
	"go.yaml.in/yaml/v4"
)

// Query represents a compiled jq query.
type Query struct {
	name string
	code *gojq.Code
}

// Error represents a jq runtime error emitted by error(v).
type Error struct {
	Value any
}

// Error formats the jq error value. If Value is a string, it returns the string
// directly; otherwise it returns the JSON representation of Value.
func (e *Error) Error() string {
	if s, ok := e.Value.(string); ok {
		return s
	}
	b, err := json.Marshal(e.Value)
	if err != nil {
		return fmt.Sprintf("%v", e.Value)
	}
	return string(b)
}

var forbiddenReasons = map[string]string{
	"now":               "it reads the clock",
	"mktime":            "it reads the clock",
	"gmtime":            "it reads the clock",
	"localtime":         "it reads the clock",
	"strftime":          "it reads the clock",
	"strflocaltime":     "it reads the clock",
	"strptime":          "it reads the clock",
	"todate":            "it reads the clock",
	"fromdate":          "it reads the clock",
	"date":              "it reads the clock",
	"dateadd":           "it reads the clock",
	"datesub":           "it reads the clock",
	"todateiso8601":     "it reads the clock",
	"fromdateiso8601":   "it reads the clock",
	"env":               "it reads the environment",
	"$ENV":              "it reads the environment",
	"input":             "it reads other input",
	"inputs":            "it reads other input",
	"input_filename":    "it reads other input",
	"input_line_number": "it reads other input",
	"debug":             "it is a side channel",
	"stderr":            "it is a side channel",
	"halt":              "it is a side channel",
	"halt_error":        "it is a side channel",
	"import":            "modules are not allowed",
	"include":           "modules are not allowed",
	"get_search_list":   "modules are not allowed",
	"modulemeta":        "modules are not allowed",
	"$__loc__":          "modules are not allowed",
}

// Compile parses defs + "\n" + src, walks the AST to refuse forbidden functions
// and variables, and compiles the query with no environment loader, no module
// loader, and no input iterator. vars names the variables, such as
// "$params", that Run binds, in order, for the whole query, defs included.
func Compile(name, defs, src string, vars ...string) (*Query, error) {
	combined := defs + "\n" + src
	parsed, err := gojq.Parse(combined)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", name, err.Error())
	}

	if item, reason := walkQuery(parsed); item != "" {
		return nil, fmt.Errorf("%s: %q is not allowed in a query (%s)", name, item, reason)
	}

	code, err := gojq.Compile(parsed, gojq.WithVariables(vars))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	return &Query{
		name: name,
		code: code,
	}, nil
}

// Run executes the compiled query against input under ctx, collecting all outputs.
// values bind the variables Compile named, in order.
// A jq error(v) stops execution and is returned as a *Error.
// Context cancellation or expiration returns an error wrapping ctx.Err().
func (q *Query) Run(ctx context.Context, input any, values ...any) ([]any, error) {
	iter := q.code.RunWithContext(ctx, input, values...)
	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil, fmt.Errorf("query timed out: %w", ctx.Err())
			}
			if ve, isVE := err.(gojq.ValueError); isVE {
				return nil, &Error{Value: ve.Value()}
			}
			return nil, err
		}
		results = append(results, v)
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("query timed out: %w", ctx.Err())
	}
	return results, nil
}

// Decode decodes data as JSON or YAML into Go types accepted by gojq.
// For JSON, trailing data after the single value is an error.
// For YAML, exactly one document is decoded.
// Numbers are converted to int when they fit, *big.Int for large integers,
// and float64 for non-integers. Mapping keys must be strings. Timestamps stay
// strings as written. .nan and .inf are refused. Aliases are expanded.
func Decode(data []byte, isYAML bool) (any, error) {
	if isYAML {
		return decodeYAML(data)
	}
	return decodeJSON(data)
}

// DecodeNode turns a node of a YAML file that yamlx has read into the JSON
// value a query sees, by Decode's rules for YAML.
func DecodeNode(node *yaml.Node) (any, error) {
	d := &yamlDecoder{onPath: map[*yaml.Node]bool{}, budget: aliasBudget(0)}
	return d.decode(node, false)
}

// Normalize turns a Go value into the JSON value a query sees: it encodes v
// as JSON and decodes it as Decode does. genxtest uses it for the values a
// test gives.
func Normalize(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return decodeJSON(b)
}

// Kind names the JSON type of a decoded value: null, boolean, number,
// string, array or object.
func Kind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "number"
	}
}

// Compact encodes a decoded value as one line of JSON, object keys sorted,
// for messages and genx list.
func Compact(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var val any
	if err := dec.Decode(&val); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("unexpected trailing data after JSON value")
		}
		return nil, err
	}
	return convertJSONValue(val)
}

func convertJSONValue(v any) (any, error) {
	switch x := v.(type) {
	case json.Number:
		return convertNumber(string(x))
	case map[string]any:
		res := make(map[string]any, len(x))
		for k, val := range x {
			c, err := convertJSONValue(val)
			if err != nil {
				return nil, err
			}
			res[k] = c
		}
		return res, nil
	case []any:
		res := make([]any, len(x))
		for i, val := range x {
			c, err := convertJSONValue(val)
			if err != nil {
				return nil, err
			}
			res[i] = c
		}
		return res, nil
	default:
		return x, nil
	}
}

func convertNumber(s string) (any, error) {
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, errors.New("NaN and Inf are not allowed")
		}
		return f, nil
	}
	if i, err := strconv.ParseInt(s, 10, 0); err == nil {
		return int(i), nil
	}
	if bi, ok := new(big.Int).SetString(s, 10); ok {
		return bi, nil
	}
	return nil, fmt.Errorf("invalid number: %s", s)
}

func decodeYAML(data []byte) (any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple YAML documents in data")
		}
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, errors.New("empty YAML document")
	}
	d := &yamlDecoder{onPath: map[*yaml.Node]bool{}, budget: aliasBudget(len(data))}
	return d.decode(&doc, false)
}

// aliasBudget is how many values a YAML file may expand to through aliases:
// enough for any real file, too few for one that expands exponentially
// (a "billion laughs" file).
func aliasBudget(size int) int { return 10000 + 100*size }

// yamlDecoder turns a YAML node tree into JSON values, expanding aliases.
// onPath holds the nodes being decoded, from the root to here: an alias to
// one of them is a cycle. A node reached twice on different paths is fine.
type yamlDecoder struct {
	onPath map[*yaml.Node]bool
	budget int
}

// resolve follows aliases to the node they name.
func resolve(node *yaml.Node) (*yaml.Node, bool, error) {
	aliased := false
	for node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil, false, errors.New("unresolved alias")
		}
		node, aliased = node.Alias, true
	}
	return node, aliased, nil
}

func (d *yamlDecoder) decode(node *yaml.Node, inAlias bool) (any, error) {
	node, aliased, err := resolve(node)
	if err != nil {
		return nil, err
	}
	inAlias = inAlias || aliased
	if inAlias {
		if d.budget--; d.budget < 0 {
			return nil, errors.New("the YAML aliases expand to too many values")
		}
	}
	if d.onPath[node] {
		return nil, errors.New("circular alias detected in YAML")
	}
	d.onPath[node] = true
	defer delete(d.onPath, node)

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return d.decode(node.Content[0], inAlias)
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!null":
			return nil, nil
		case "!!bool":
			return strconv.ParseBool(node.Value)
		case "!!int":
			return convertNumber(node.Value)
		case "!!float":
			lower := strings.ToLower(node.Value)
			if strings.Contains(lower, "nan") {
				return nil, errors.New(".nan is not allowed in YAML")
			}
			if strings.Contains(lower, "inf") {
				return nil, errors.New(".inf is not allowed in YAML")
			}
			return convertNumber(node.Value)
		default: // !!str, !!timestamp (kept as written) and other tags
			return node.Value, nil
		}
	case yaml.SequenceNode:
		res := make([]any, len(node.Content))
		for i, elem := range node.Content {
			v, err := d.decode(elem, inAlias)
			if err != nil {
				return nil, err
			}
			res[i] = v
		}
		return res, nil
	case yaml.MappingNode:
		res := make(map[string]any)
		var merges []*yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			kNode, vNode := node.Content[i], node.Content[i+1]
			if kNode.Value == "<<" && kNode.Kind == yaml.ScalarNode {
				merges = append(merges, vNode)
				continue
			}
			kNode, _, err := resolve(kNode)
			if err != nil {
				return nil, err
			}
			if kNode.Kind != yaml.ScalarNode || (kNode.Tag != "!!str" && kNode.Tag != "") {
				return nil, fmt.Errorf("mapping keys must be strings: found tag %s", kNode.Tag)
			}
			val, err := d.decode(vNode, inAlias)
			if err != nil {
				return nil, err
			}
			res[kNode.Value] = val
		}
		// A merge key (<<) adds the keys of a mapping, or of each mapping in
		// a sequence, that the mapping doesn't set itself.
		for _, m := range merges {
			target, _, err := resolve(m)
			if err != nil {
				return nil, err
			}
			sources := []*yaml.Node{m}
			if target.Kind == yaml.SequenceNode {
				sources = target.Content
			}
			for _, src := range sources {
				v, err := d.decode(src, inAlias)
				if err != nil {
					return nil, err
				}
				if mm, ok := v.(map[string]any); ok {
					for k, val := range mm {
						if _, set := res[k]; !set {
							res[k] = val
						}
					}
				}
			}
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unknown YAML node kind: %v", node.Kind)
	}
}

// Encode encodes v as JSON with two-space indent, object keys sorted, exact integers,
// no HTML escaping, and a final newline.
func Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// walkQuery finds the first refused function, variable or directive in q. It
// walks every exported field of the parsed query by reflection, so a part of
// gojq's syntax tree it doesn't name can't hide a call.
func walkQuery(q *gojq.Query) (item, reason string) {
	var walk func(v reflect.Value) bool
	walk = func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return false
			}
			name := ""
			switch x := v.Interface().(type) {
			case *gojq.Import:
				name = "import"
				if x.IncludePath != "" {
					name = "include"
				}
			case *gojq.Func:
				name = x.Name
			case *gojq.FuncDef:
				name = x.Name
				for _, arg := range x.Args {
					if _, ok := forbiddenReasons[arg]; ok {
						name = arg
					}
				}
			case *gojq.Pattern:
				name = x.Name
			case *gojq.ObjectKeyVal:
				name = x.Key // {$ENV} is shorthand for {ENV: $ENV}
			case *gojq.PatternObject:
				name = x.Key
			}
			if r, ok := forbiddenReasons[name]; ok {
				item, reason = name, r
				return true
			}
			return walk(v.Elem())
		case reflect.Struct:
			for i := range v.NumField() {
				if v.Type().Field(i).IsExported() && walk(v.Field(i)) {
					return true
				}
			}
		case reflect.Slice:
			for i := range v.Len() {
				if walk(v.Index(i)) {
					return true
				}
			}
		}
		return false
	}
	walk(reflect.ValueOf(q))
	return item, reason
}
