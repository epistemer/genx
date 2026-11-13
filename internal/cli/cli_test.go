package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestMainExitCodes(t *testing.T) {
	tests := []struct {
		args       []string
		code       int
		stdout     string
		stderrPart string
	}{
		{args: []string{"help"}, code: 0, stdout: usage},
		{args: []string{"-h"}, code: 0, stdout: usage},
		{args: []string{"frobnicate"}, code: 2, stderrPart: `genx: unknown command "frobnicate"`},
		{args: []string{"check", "x"}, code: 2, stderrPart: "genx check: takes no arguments"},
		{args: []string{"list", "x"}, code: 2, stderrPart: "genx list: takes no arguments"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := Main(tt.args, &stdout, &stderr); got != tt.code {
				t.Errorf("exit code = %d, want %d", got, tt.code)
			}
			if stdout.String() != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.stdout)
			}
			if !strings.Contains(stderr.String(), tt.stderrPart) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.stderrPart)
			}
		})
	}
}
