package manager

import (
	"reflect"
	"testing"
)

func TestPacmanPrintDeps(t *testing.T) {
	out := []byte("asciiquarium\nperl-curses\nperl-term-animation\n")
	got := pacmanPrintDeps(out, "asciiquarium")
	want := []string{"perl-curses", "perl-term-animation"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pacmanPrintDeps = %v, want %v", got, want)
	}
	// A target with no extra dependencies yields nothing.
	if got := pacmanPrintDeps([]byte("htop\n"), "htop"); len(got) != 0 {
		t.Errorf("expected no deps, got %v", got)
	}
}
