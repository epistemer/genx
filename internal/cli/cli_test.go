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
		{args: nil, code: 1, stderrPart: "genx: not implemented yet"},
		{args: []string{"check"}, code: 1, stderrPart: "genx check: not implemented yet"},
		{args: []string{"list"}, code: 1, stderrPart: "genx list: not implemented yet"},
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
