package cli

import (
	"reflect"
	"testing"
)

func TestParseSelection(t *testing.T) {
	cases := []struct {
		in      string
		n       int
		want    []int
		wantErr bool
	}{
		{"", 5, nil, false},
		{"all", 5, []int{0, 1, 2, 3, 4}, false},
		{"1 3", 5, []int{0, 2}, false},
		{"2-4", 5, []int{1, 2, 3}, false},
		{"1,4", 5, []int{0, 3}, false},
		{"4-2", 5, []int{1, 2, 3}, false},
		{"1 1", 5, []int{0}, false},
		{"9", 5, nil, true},
		{"x", 5, nil, true},
		{"1-x", 5, nil, true},
	}
	for _, c := range cases {
		got, err := parseSelection(c.in, c.n)
		if (err != nil) != c.wantErr {
			t.Errorf("parseSelection(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseSelection(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
