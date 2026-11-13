package domain

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Domain
		wantErr string
	}{
		// Valid cases
		{input: "labels/v1", want: Domain{Name: "labels", Major: 1}},
		{input: "sqlc.request/v1", want: Domain{Name: "sqlc.request", Major: 1}},
		{input: "a.b-c.d2/v42", want: Domain{Name: "a.b-c.d2", Major: 42}},
		{input: "z/v999", want: Domain{Name: "z", Major: 999}},

		// Invalid cases
		{input: "", wantErr: `"": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels", wantErr: `"labels": invalid domain (want name/vN, such as labels/v1)`},
		{input: "/v1", wantErr: `"/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/", wantErr: `"labels/": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/v", wantErr: `"labels/v": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/v0", wantErr: `"labels/v0": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/v01", wantErr: `"labels/v01": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/v-1", wantErr: `"labels/v-1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels/1", wantErr: `"labels/1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "Labels/v1", wantErr: `"Labels/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels./v1", wantErr: `"labels./v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: ".labels/v1", wantErr: `".labels/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "a..b/v1", wantErr: `"a..b/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "a/b/v1", wantErr: `"a/b/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "-labels/v1", wantErr: `"-labels/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "0labels/v1", wantErr: `"0labels/v1": invalid domain (want name/vN, such as labels/v1)`},
		{input: "labels_v1/v1", wantErr: `"labels_v1/v1": invalid domain (want name/vN, such as labels/v1)`},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Parse(tc.input)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %v", tc.input, err)
				}
				if got != tc.want {
					t.Fatalf("Parse(%q) = %+v, want %+v", tc.input, got, tc.want)
				}
				if got.String() != tc.input {
					t.Fatalf("Domain.String() = %q, want %q", got.String(), tc.input)
				}
			} else {
				if err == nil {
					t.Fatalf("Parse(%q) expected error %q, got nil", tc.input, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("Parse(%q) error = %q, want %q", tc.input, err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestDomainString(t *testing.T) {
	var zero Domain
	if s := zero.String(); s != "" {
		t.Errorf("zero.String() = %q, want %q", s, "")
	}
	d := Domain{Name: "labels", Major: 1}
	if s := d.String(); s != "labels/v1" {
		t.Errorf("d.String() = %q, want %q", s, "labels/v1")
	}
}
