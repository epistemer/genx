package query

import "testing"

// TestRefusedAnywhere puts refused names in every corner of jq's syntax.
func TestRefusedAnywhere(t *testing.T) {
	for _, src := range []string{`{$ENV}`, `{$__loc__}`, `. as {$ENV} | 1`, `[.[] | {a: input}]`, `.a as [$x] ?// $x | now`, `limit(1; now)`, `first(debug)`, `path(..) | stderr`, `1 as $x | now`, `. as [$a] | $ENV`, `.[] as {a: $v} | input`, `reduce .[] as $x (0; . + now)`, `label $out | now`, `try now catch .`, `{(now|tostring): 1}`, `"\(now)"`, `.[now]`, `def f: now; f`, `$__loc__`, `@base64 "\(env)"`, `foreach .[] as $x (0; now; .)`, `if . then 1 elif now then 2 else 3 end`, `.[1:now]`} {
		if _, err := Compile("q", "", src); err == nil {
			t.Errorf("%s: compiled, want it refused", src)
		}
	}
}
