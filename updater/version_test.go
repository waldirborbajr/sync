package updater

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		remote  string
		want    bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"1.0.0", "1.1.0", true},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "1.0.1", true},
		{"1.0.0", "v1.0.1", true},
		{"", "1.0.0", false},
		{"1.0.0", "", false},
		{"1.0", "1.0.1", true}, // missing patch
		{"1.0.0", "1.0", false},
		{"1.0.0", "1.0.0.1", false}, // extra parts ignored
	}

	for _, tt := range tests {
		got := isNewerVersion(tt.current, tt.remote)
		if got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v; want %v", tt.current, tt.remote, got, tt.want)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{" v1.0.0 ", "1.0.0"},
		{"v", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}
