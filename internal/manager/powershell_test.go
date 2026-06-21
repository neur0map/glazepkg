package manager

import (
	"encoding/json"
	"testing"
)

func TestPsModuleScopeUnmarshal(t *testing.T) {
	data := []byte(`[{"Name":"PSReadLine","Version":"2.3.4","Description":"","Scope":"AllUsers"},{"Name":"MyMod","Version":"1.0.0","Description":"","Scope":"CurrentUser"}]`)
	var mods []psModule
	if err := json.Unmarshal(data, &mods); err != nil {
		t.Fatal(err)
	}
	if mods[0].Scope != "AllUsers" || mods[1].Scope != "CurrentUser" {
		t.Errorf("scopes: %q %q", mods[0].Scope, mods[1].Scope)
	}
}
