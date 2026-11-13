package paths

import (
	"testing"
)

func TestFold(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"Foo/Bar/ZZ_out.go", "foo/bar/zz_out.go"},
		{"abc", "abc"},
		{"XYZ_123.JSON", "xyz_123.json"},
	}
	for _, tc := range tests {
		got := Fold(tc.input)
		if got != tc.want {
			t.Errorf("Fold(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDeclared(t *testing.T) {
	tests := []struct {
		path    string
		wantErr string
	}{
		{path: "models/codegen_request.json", wantErr: ""},
		{path: "internal/settings/zz_settings.labels.json", wantErr: ""},
		{path: "a/b/c.go", wantErr: ""},
		{path: "file.txt", wantErr: ""},
		// Not clean
		{path: "a/../b", wantErr: `"a/../b": not clean (use "b")`},
		{path: "./a", wantErr: `"./a": not clean (use "a")`},
		{path: "a//b", wantErr: `"a//b": not clean (use "a/b")`},
		{path: "a/", wantErr: `"a/": not clean (use "a")`},
		{path: "", wantErr: `"": not clean (use ".")`},
		// Must not be "."
		{path: ".", wantErr: `".": must not be "."`},
		// Absolute
		{path: "/a/b", wantErr: `"/a/b": must not be absolute`},
		// Contains ".."
		{path: "../b", wantErr: `"../b": must not contain ".."`},
		// Starts/ends with '.'
		{path: ".hidden", wantErr: `".hidden": element ".hidden" must not start with '.'`},
		{path: "a/.b", wantErr: `"a/.b": element ".b" must not start with '.'`},
		{path: "bad.", wantErr: `"bad.": element "bad." must not end with '.'`},
		{path: "a/b.", wantErr: `"a/b.": element "b." must not end with '.'`},
		// Invalid characters
		{path: "a/b@c", wantErr: `"a/b@c": element "b@c" contains invalid character '@'`},
		{path: "a/b:c", wantErr: `"a/b:c": element "b:c" contains invalid character ':'`},
		{path: "a/b c", wantErr: `"a/b c": element "b c" contains invalid character ' '`},
		// Windows reserved names
		{path: "x/CON.go", wantErr: `"x/CON.go": "CON" is a reserved name on Windows`},
		{path: "con", wantErr: `"con": "con" is a reserved name on Windows`},
		{path: "a/prn.txt", wantErr: `"a/prn.txt": "prn" is a reserved name on Windows`},
		{path: "AUX.json", wantErr: `"AUX.json": "AUX" is a reserved name on Windows`},
		{path: "sub/NUL.tar.gz", wantErr: `"sub/NUL.tar.gz": "NUL" is a reserved name on Windows`},
		{path: "com1.out", wantErr: `"com1.out": "com1" is a reserved name on Windows`},
		{path: "COM9.dat", wantErr: `"COM9.dat": "COM9" is a reserved name on Windows`},
		{path: "lpt1.txt", wantErr: `"lpt1.txt": "lpt1" is a reserved name on Windows`},
		{path: "LPT9", wantErr: `"LPT9": "LPT9" is a reserved name on Windows`},
		// Valid names similar to reserved names
		{path: "connect.go", wantErr: ""},
		{path: "com0.txt", wantErr: ""},
		{path: "com10.txt", wantErr: ""},
		{path: "lpt0.txt", wantErr: ""},
		{path: "lpt10.txt", wantErr: ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			err := Declared(tc.path)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Declared(%q) unexpected error: %v", tc.path, err)
				}
			} else {
				if err == nil {
					t.Fatalf("Declared(%q) expected error %q, got nil", tc.path, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("Declared(%q) error = %q, want %q", tc.path, err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestDir(t *testing.T) {
	tests := []struct {
		dir     string
		wantErr string
	}{
		{dir: ".", wantErr: ""},
		{dir: "internal/settings", wantErr: ""},
		{dir: "pkg/gen", wantErr: ""},
		{dir: "a/../b", wantErr: `"a/../b": not clean (use "b")`},
		{dir: "/abs/path", wantErr: `"/abs/path": must not be absolute`},
		{dir: "x/CON", wantErr: `"x/CON": "CON" is a reserved name on Windows`},
	}
	for _, tc := range tests {
		t.Run(tc.dir, func(t *testing.T) {
			err := Dir(tc.dir)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Dir(%q) unexpected error: %v", tc.dir, err)
				}
			} else {
				if err == nil {
					t.Fatalf("Dir(%q) expected error %q, got nil", tc.dir, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("Dir(%q) error = %q, want %q", tc.dir, err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestOut(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr string
	}{
		{path: "zz_out.go", want: "zz_out.go", wantErr: ""},
		{path: "./zz_out.go", want: "zz_out.go", wantErr: ""},
		{path: "models/zz_enums.labels.json", want: "models/zz_enums.labels.json", wantErr: ""},
		{path: "internal/settings/./zz_settings_i18n.go", want: "internal/settings/zz_settings_i18n.go", wantErr: ""},
		// Missing zz_ prefix
		{path: "x/a.go", wantErr: `"x/a.go": the base name must start with zz_`},
		{path: "out.go", wantErr: `"out.go": the base name must start with zz_`},
		// Must not be "."
		{path: ".", wantErr: `".": must not be "."`},
		{path: "", wantErr: `"": must not be "."`},
		// Absolute
		{path: "/zz_a.go", wantErr: `"/zz_a.go": must not be absolute`},
		// Windows reserved name
		{path: "CON/zz_a.go", wantErr: `"CON/zz_a.go": "CON" is a reserved name on Windows`},
		// Invalid characters
		{path: "pkg/zz_a@b.go", wantErr: `"pkg/zz_a@b.go": element "zz_a@b.go" contains invalid character '@'`},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, err := Out(tc.path)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Out(%q) unexpected error: %v", tc.path, err)
				}
				if got != tc.want {
					t.Fatalf("Out(%q) = %q, want %q", tc.path, got, tc.want)
				}
			} else {
				if err == nil {
					t.Fatalf("Out(%q) expected error %q, got nil", tc.path, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("Out(%q) error = %q, want %q", tc.path, err.Error(), tc.wantErr)
				}
			}
		})
	}
}
