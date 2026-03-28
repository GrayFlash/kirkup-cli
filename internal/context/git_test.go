package context

import (
	"testing"
)

func TestNormaliseRemote(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"git@github.com:example/repo.git", "github.com/example/repo"},
		{"https://github.com/example/repo.git", "github.com/example/repo"},
		{"https://github.com/example/repo", "github.com/example/repo"},
		{"http://github.com/example/repo.git", "github.com/example/repo"},
		{"github.com/example/repo", "github.com/example/repo"},
		{"", ""},
	}

	for _, c := range cases {
		got := normaliseRemote(c.input)
		if got != c.want {
			t.Errorf("normaliseRemote(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
