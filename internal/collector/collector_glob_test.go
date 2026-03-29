package collector

import (
	"testing"
)

func TestGlobWatchDir(t *testing.T) {
	cases := []struct {
		pattern string
		want    string
	}{
		{"/home/user/.gemini/tmp/*/logs.json", "/home/user/.gemini/tmp"},
		{"/home/user/.config/Cursor/User/workspaceStorage/*/state.vscdb", "/home/user/.config/Cursor/User/workspaceStorage"},
		{"/no/wildcards/file.json", "/no/wildcards"},
		{"/path/to/?.json", "/path/to"},
	}

	for _, c := range cases {
		got := globWatchDir(c.pattern)
		if got != c.want {
			t.Errorf("globWatchDir(%q) = %q, want %q", c.pattern, got, c.want)
		}
	}
}
