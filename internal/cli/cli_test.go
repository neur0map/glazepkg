package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"bogus"}, nil, "test", &out, &errOut)
	if code != ExitErr {
		t.Errorf("exit code = %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "unknown subcommand") {
		t.Errorf("stderr = %q, want substring 'unknown subcommand'", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestDispatchEmptyArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch(nil, nil, "test", &out, &errOut)
	if code != ExitErr {
		t.Errorf("exit code = %d, want %d", code, ExitErr)
	}
}

func TestExitCodeConstants(t *testing.T) {
	if ExitOK != 0 {
		t.Errorf("ExitOK = %d, want 0", ExitOK)
	}
	if ExitErr != 1 {
		t.Errorf("ExitErr = %d, want 1", ExitErr)
	}
	if ExitNegative != 2 {
		t.Errorf("ExitNegative = %d, want 2", ExitNegative)
	}
}
