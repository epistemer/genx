package render

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/scanner"
	"regexp"
	"strings"
	"text/template"
)

// Template represents a parsed Go text template with strict missing-key behavior.
type Template struct {
	name string
	tmpl *template.Template
}

var headerRegexp = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// Parse parses a template from src with missingkey=error, no Funcs, and CRLF
// converted to LF. name is the template file name used in error messages.
func Parse(name string, src []byte) (*Template, error) {
	normalized := bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(normalized))
	if err != nil {
		return nil, err
	}
	return &Template{
		name: name,
		tmpl: tmpl,
	}, nil
}

// Execute executes the template with unit as its data, returning the rendered
// bytes. Errors retain text/template's message including name and line number.
func (t *Template) Execute(unit any) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, unit); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HasHeader reports whether the first line of b (with any trailing CR stripped)
// matches the generated-code header regexp ^// Code generated .* DO NOT EDIT\.$
func HasHeader(b []byte) bool {
	firstLine := b
	if idx := bytes.IndexByte(b, '\n'); idx != -1 {
		firstLine = b[:idx]
	}
	firstLine = bytes.TrimSuffix(firstLine, []byte("\r"))
	return headerRegexp.Match(firstLine)
}

// Finish formats the rendered output. If b contains only whitespace, empty is true.
// If outPath ends in .go, b is formatted with go/format.Source; on failure the error
// displays the format error and the generated lines around the error line (3 before,
// 3 after, numbered). After formatting, the first line must match the generated Go
// header regexp, or Finish returns an error. Non-.go output has its CRLF line
// endings turned into LF (interface.md §2).
func Finish(outPath string, b []byte) (out []byte, empty bool, err error) {
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, true, nil
	}
	if !strings.HasSuffix(outPath, ".go") {
		return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), false, nil
	}

	formatted, err := format.Source(b)
	if err != nil {
		errLine := 1
		if el, ok := err.(scanner.ErrorList); ok && len(el) > 0 {
			errLine = el[0].Pos.Line
		} else if se, ok := err.(scanner.Error); ok {
			errLine = se.Pos.Line
		} else {
			var l int
			if n, _ := fmt.Sscanf(err.Error(), "%d:", &l); n == 1 {
				errLine = l
			}
		}

		lines := strings.Split(string(b), "\n")
		start := errLine - 3
		if start < 1 {
			start = 1
		}
		end := errLine + 3
		if end > len(lines) {
			end = len(lines)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s: %v\n", outPath, err))
		for i := start; i <= end; i++ {
			sb.WriteString(fmt.Sprintf("%4d: %s\n", i, lines[i-1]))
		}
		return nil, false, errors.New(strings.TrimRight(sb.String(), "\n"))
	}

	if !HasHeader(formatted) {
		return nil, false, fmt.Errorf("%s: a generated Go file must start with %q", outPath, "// Code generated ... DO NOT EDIT.")
	}

	return formatted, false, nil
}
