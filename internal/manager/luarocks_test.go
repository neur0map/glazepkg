package manager

import "testing"

func TestLuarocksScope(t *testing.T) {
	home := "/home/u"
	cases := []struct {
		path, want string
	}{
		{"/home/u/.luarocks/lib/luarocks/rocks-5.4", "user"},
		{"/usr/local/lib/luarocks/rocks-5.4", "system"},
		{"", ""},
	}
	for _, c := range cases {
		if got := luarocksScope(c.path, home); got != c.want {
			t.Errorf("luarocksScope(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
