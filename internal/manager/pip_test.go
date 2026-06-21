package manager

import "testing"

func TestNormalizePipName(t *testing.T) {
	if got := normalizePipName("Flask_SQLAlchemy"); got != "flask-sqlalchemy" {
		t.Errorf("normalizePipName = %q", got)
	}
}

func TestParsePipNameSet(t *testing.T) {
	data := []byte(`[{"name":"Black","version":"24.3.0"},{"name":"requests_oauthlib","version":"1.0"}]`)
	set := parsePipNameSet(data)
	if !set["black"] || !set["requests-oauthlib"] {
		t.Errorf("set missing normalized entries: %v", set)
	}
	if parsePipNameSet([]byte("not json")) != nil {
		t.Error("expected nil on bad json")
	}
}

func TestPipScope(t *testing.T) {
	user := map[string]bool{"black": true}
	if pipScope("Black", user) != "user" {
		t.Error("Black should be user scope")
	}
	if pipScope("setuptools", user) != "global" {
		t.Error("setuptools should be global scope")
	}
	if pipScope("black", nil) != "" {
		t.Error("nil set should yield empty scope")
	}
}
