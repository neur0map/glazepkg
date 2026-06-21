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

func TestSumPacmanSizes(t *testing.T) {
	out := []byte(`Name            : perl-curses
Download Size   : 200.00 KiB
Installed Size  : 500.00 KiB

Name            : perl-term-animation
Download Size   : 40.00 KiB
Installed Size  : 290.00 KiB
`)
	dl, inst := sumPacmanSizes(out)
	if dl != 240*1024 {
		t.Errorf("download = %d, want %d", dl, 240*1024)
	}
	if inst != 790*1024 {
		t.Errorf("installed = %d, want %d", inst, 790*1024)
	}
}
