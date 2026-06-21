package cli

import (
	"reflect"
	"testing"
)

func TestTranslateOps(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
		ok   bool
	}{
		{[]string{"-S", "ffmpeg"}, []string{"install", "ffmpeg"}, true},
		{[]string{"-Ss", "ripgrep"}, []string{"search", "ripgrep"}, true},
		{[]string{"-Si", "vim"}, []string{"info", "vim"}, true},
		{[]string{"-Syu"}, []string{"upgrade"}, true},
		{[]string{"-Su"}, []string{"upgrade"}, true},
		{[]string{"-Sy"}, []string{"refresh"}, true},
		{[]string{"-Syy"}, []string{"refresh"}, true},
		{[]string{"-Sy", "--manager", "pacman"}, []string{"refresh", "--manager", "pacman"}, true},
		{[]string{"-Sy", "htop"}, []string{"install", "htop"}, true},
		{[]string{"-Sc"}, []string{"clean"}, true},
		{[]string{"-Scc"}, []string{"clean", "--all"}, true},
		{[]string{"-R", "foo"}, []string{"remove", "foo"}, true},
		{[]string{"-Rs", "foo"}, []string{"remove", "--with-deps", "foo"}, true},
		{[]string{"-Rns", "foo"}, []string{"remove", "--with-deps", "foo"}, true},
		{[]string{"-Q"}, []string{"list"}, true},
		{[]string{"-Qi", "foo"}, []string{"info", "foo"}, true},
		{[]string{"-Qs", "term"}, []string{"list", "term"}, true},
		{[]string{"-Qu"}, []string{"outdated"}, true},
		{[]string{"-Qdt"}, []string{"autoremove", "--print"}, true},
		{[]string{"-S", "pkg", "--noconfirm"}, []string{"install", "pkg", "--yes"}, true},
		{[]string{"install", "foo"}, nil, false},
		{[]string{"--manager", "x"}, nil, false},
		{[]string{"foo"}, nil, false},
	}
	for _, c := range cases {
		got, ok := TranslateOps(c.in)
		if ok != c.ok {
			t.Errorf("TranslateOps(%v) ok=%v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && !reflect.DeepEqual(got, c.want) {
			t.Errorf("TranslateOps(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
