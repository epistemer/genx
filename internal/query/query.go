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
// loader, and no input iterator.
func Compile(name, defs, src string) (*Query, error) {
	combined := defs + "\n" + src
	parsed, err := gojq.Parse(combined)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", name, err.Error())
	}

	if item, reason := walkQuery(parsed); item != "" {
		return nil, fmt.Errorf("%s: %q is not allowed in a query (%s)", name, item, reason)
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	return &Query{
		name: name,
		code: code,
	}, nil
}

// Run executes the compiled query against input under ctx, collecting all outputs.
// A jq error(v) stops execution and is returned as a *Error.
// Context cancellation or expiration returns an error wrapping ctx.Err().
func (q *Query) Run(ctx context.Context, input any) ([]any, error) {
	iter := q.code.RunWithContext(ctx, input)
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
	return decodeYAMLNode(&doc, make(map[*yaml.Node]bool))
}

func decodeYAMLNode(node *yaml.Node, seen map[*yaml.Node]bool) (any, error) {
	for node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil, errors.New("unresolved alias")
		}
		if seen[node] {
			return nil, errors.New("circular alias detected in YAML")
		}
		seen[node] = true
		node = node.Alias
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return decodeYAMLNode(node.Content[0], seen)
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
		case "!!timestamp":
			return node.Value, nil
		case "!!str", "":
			return node.Value, nil
		default:
			return node.Value, nil
		}
	case yaml.SequenceNode:
		res := make([]any, len(node.Content))
		for i, elem := range node.Content {
			v, err := decodeYAMLNode(elem, seen)
			if err != nil {
				return nil, err
			}
			res[i] = v
		}
		return res, nil
	case yaml.MappingNode:
		res := make(map[string]any)
		for i := 0; i < len(node.Content); i += 2 {
			kNode := node.Content[i]
			vNode := node.Content[i+1]

			if kNode.Value == "<<" {
				target := vNode
				for target.Kind == yaml.AliasNode {
					if seen[target] {
						return nil, errors.New("circular alias in merge key")
					}
					seen[target] = true
					target = target.Alias
				}
				if target.Kind == yaml.MappingNode {
					merged, err := decodeYAMLNode(target, seen)
					if err != nil {
						return nil, err
					}
					if mMap, ok := merged.(map[string]any); ok {
						for mk, mv := range mMap {
							if _, exists := res[mk]; !exists {
								res[mk] = mv
							}
						}
					}
				} else if target.Kind == yaml.SequenceNode {
					for _, item := range target.Content {
						for item.Kind == yaml.AliasNode {
							if seen[item] {
								return nil, errors.New("circular alias in merge key sequence")
							}
							seen[item] = true
							item = item.Alias
						}
						if item.Kind == yaml.MappingNode {
							merged, err := decodeYAMLNode(item, seen)
							if err != nil {
								return nil, err
							}
							if mMap, ok := merged.(map[string]any); ok {
								for mk, mv := range mMap {
									if _, exists := res[mk]; !exists {
										res[mk] = mv
									}
								}
							}
						}
					}
				}
				continue
			}

			for kNode.Kind == yaml.AliasNode {
				kNode = kNode.Alias
			}
			if kNode.Kind != yaml.ScalarNode || (kNode.Tag != "!!str" && kNode.Tag != "") {
				return nil, fmt.Errorf("mapping keys must be strings: found tag %s", kNode.Tag)
			}
			val, err := decodeYAMLNode(vNode, seen)
			if err != nil {
				return nil, err
			}
			res[kNode.Value] = val
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

func walkQuery(q *gojq.Query) (string, string) {
	if q == nil {
		return "", ""
	}
	for _, im := range q.Imports {
		if im.IncludePath != "" {
			return "include", forbiddenReasons["include"]
		}
		return "import", forbiddenReasons["import"]
	}
	for _, fd := range q.FuncDefs {
		if reason, ok := forbiddenReasons[fd.Name]; ok {
			return fd.Name, reason
		}
		for _, arg := range fd.Args {
			if reason, ok := forbiddenReasons[arg]; ok {
				return arg, reason
			}
		}
		if item, reason := walkQuery(fd.Body); item != "" {
			return item, reason
		}
	}
	if item, reason := walkQuery(q.Left); item != "" {
		return item, reason
	}
	for _, p := range q.Patterns {
		if item, reason := walkPattern(p); item != "" {
			return item, reason
		}
	}
	if item, reason := walkQuery(q.Right); item != "" {
		return item, reason
	}
	if q.Term != nil {
		if item, reason := walkTerm(q.Term); item != "" {
			return item, reason
		}
	}
	return "", ""
}

func walkPattern(p *gojq.Pattern) (string, string) {
	if p == nil {
		return "", ""
	}
	if reason, ok := forbiddenReasons[p.Name]; ok {
		return p.Name, reason
	}
	for _, elem := range p.Array {
		if item, reason := walkPattern(elem); item != "" {
			return item, reason
		}
	}
	for _, obj := range p.Object {
		if reason, ok := forbiddenReasons[obj.Key]; ok {
			return obj.Key, reason
		}
		if obj.KeyString != nil {
			for _, sq := range obj.KeyString.Queries {
				if item, reason := walkQuery(sq); item != "" {
					return item, reason
				}
			}
		}
		if obj.KeyQuery != nil {
			if item, reason := walkQuery(obj.KeyQuery); item != "" {
				return item, reason
			}
		}
		if obj.Val != nil {
			if item, reason := walkPattern(obj.Val); item != "" {
				return item, reason
			}
		}
	}
	return "", ""
}

func walkTerm(t *gojq.Term) (string, string) {
	if t == nil {
		return "", ""
	}
	for _, s := range t.SuffixList {
		if s.Index != nil {
			if item, reason := walkIndex(s.Index); item != "" {
				return item, reason
			}
		}
	}
	switch t.Type {
	case gojq.TermTypeFunc:
		if t.Func != nil {
			if reason, ok := forbiddenReasons[t.Func.Name]; ok {
				return t.Func.Name, reason
			}
			for _, arg := range t.Func.Args {
				if item, reason := walkQuery(arg); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeFormat:
		name := strings.TrimPrefix(t.Format, "@")
		if reason, ok := forbiddenReasons[name]; ok {
			return name, reason
		}
		if t.Str != nil {
			for _, sq := range t.Str.Queries {
				if item, reason := walkQuery(sq); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeIndex:
		if item, reason := walkIndex(t.Index); item != "" {
			return item, reason
		}
	case gojq.TermTypeObject:
		if t.Object != nil {
			for _, kv := range t.Object.KeyVals {
				if kv.KeyQuery != nil {
					if item, reason := walkQuery(kv.KeyQuery); item != "" {
						return item, reason
					}
				}
				if kv.KeyString != nil {
					for _, sq := range kv.KeyString.Queries {
						if item, reason := walkQuery(sq); item != "" {
							return item, reason
						}
					}
				}
				if kv.Val != nil {
					if item, reason := walkQuery(kv.Val); item != "" {
						return item, reason
					}
				}
			}
		}
	case gojq.TermTypeArray:
		if t.Array != nil && t.Array.Query != nil {
			if item, reason := walkQuery(t.Array.Query); item != "" {
				return item, reason
			}
		}
	case gojq.TermTypeUnary:
		if t.Unary != nil && t.Unary.Term != nil {
			if item, reason := walkTerm(t.Unary.Term); item != "" {
				return item, reason
			}
		}
	case gojq.TermTypeString:
		if t.Str != nil {
			for _, sq := range t.Str.Queries {
				if item, reason := walkQuery(sq); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeIf:
		if t.If != nil {
			if item, reason := walkQuery(t.If.Cond); item != "" {
				return item, reason
			}
			if item, reason := walkQuery(t.If.Then); item != "" {
				return item, reason
			}
			for _, elif := range t.If.Elif {
				if item, reason := walkQuery(elif.Cond); item != "" {
					return item, reason
				}
				if item, reason := walkQuery(elif.Then); item != "" {
					return item, reason
				}
			}
			if t.If.Else != nil {
				if item, reason := walkQuery(t.If.Else); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeTry:
		if t.Try != nil {
			if item, reason := walkQuery(t.Try.Body); item != "" {
				return item, reason
			}
			if t.Try.Catch != nil {
				if item, reason := walkQuery(t.Try.Catch); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeReduce:
		if t.Reduce != nil {
			if item, reason := walkQuery(t.Reduce.Query); item != "" {
				return item, reason
			}
			for _, p := range t.Reduce.Patterns {
				if item, reason := walkPattern(p); item != "" {
					return item, reason
				}
			}
			if item, reason := walkQuery(t.Reduce.Start); item != "" {
				return item, reason
			}
			if item, reason := walkQuery(t.Reduce.Update); item != "" {
				return item, reason
			}
		}
	case gojq.TermTypeForeach:
		if t.Foreach != nil {
			if item, reason := walkQuery(t.Foreach.Query); item != "" {
				return item, reason
			}
			for _, p := range t.Foreach.Patterns {
				if item, reason := walkPattern(p); item != "" {
					return item, reason
				}
			}
			if item, reason := walkQuery(t.Foreach.Start); item != "" {
				return item, reason
			}
			if item, reason := walkQuery(t.Foreach.Update); item != "" {
				return item, reason
			}
			if t.Foreach.Extract != nil {
				if item, reason := walkQuery(t.Foreach.Extract); item != "" {
					return item, reason
				}
			}
		}
	case gojq.TermTypeLabel:
		if t.Label != nil {
			if reason, ok := forbiddenReasons[t.Label.Ident]; ok {
				return t.Label.Ident, reason
			}
			if item, reason := walkQuery(t.Label.Body); item != "" {
				return item, reason
			}
		}
	case gojq.TermTypeBreak:
		if reason, ok := forbiddenReasons[t.Break]; ok {
			return t.Break, reason
		}
	case gojq.TermTypeQuery:
		if t.Query != nil {
			if item, reason := walkQuery(t.Query); item != "" {
				return item, reason
			}
		}
	}
	return "", ""
}

func walkIndex(idx *gojq.Index) (string, string) {
	if idx == nil {
		return "", ""
	}
	if idx.Str != nil {
		for _, sq := range idx.Str.Queries {
			if item, reason := walkQuery(sq); item != "" {
				return item, reason
			}
		}
	}
	if idx.Start != nil {
		if item, reason := walkQuery(idx.Start); item != "" {
			return item, reason
		}
	}
	if idx.End != nil {
		if item, reason := walkQuery(idx.End); item != "" {
			return item, reason
		}
	}
	return "", ""
}
