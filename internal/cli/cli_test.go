package cli

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestDispatchUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"bogus"}, nil, "test", &out, &errOut, nil)
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
	code := Dispatch(nil, nil, "test", &out, &errOut, nil)
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

func TestDispatchRoutesToRegisteredSubcommand(t *testing.T) {
	var (
		gotArgs    []string
		gotVersion string
	)
	subcommands["__test_echo"] = func(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
		gotArgs = args
		gotVersion = version
		return 42
	}
	defer delete(subcommands, "__test_echo")

	var out, errOut bytes.Buffer
	code := Dispatch([]string{"__test_echo", "a", "b"}, nil, "v1", &out, &errOut, nil)
	if code != 42 {
		t.Errorf("exit code = %d, want 42 (handler's return value should propagate)", code)
	}
	if !reflect.DeepEqual(gotArgs, []string{"a", "b"}) {
		t.Errorf("handler got args = %v, want [a b] (Dispatch should strip args[0])", gotArgs)
	}
	if gotVersion != "v1" {
		t.Errorf("handler got version = %q, want %q", gotVersion, "v1")
	}
}
