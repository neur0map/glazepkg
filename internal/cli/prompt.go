package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// canPrompt reports whether interactive prompts make sense: a real reader is
// present and stdout is a terminal. Scripts and pipes skip menus and fall back
// to deterministic behavior instead of hanging.
func canPrompt(r *bufio.Reader) bool {
	return r != nil && colorEnabled()
}

func newPromptReader(in io.Reader) *bufio.Reader {
	if in == nil {
		return nil
	}
	return bufio.NewReader(in)
}

// answer is the outcome of a confirmation prompt. Saying no and having nobody
// there to answer are different facts: a decision, versus a run that must not
// look like a successful no-op.
type answer int

const (
	answerNo answer = iota
	answerYes
	answerUnavailable
)

// confirm prints prompt and reads a y/n answer. A shared reader is used so a
// later read in the same flow doesn't lose buffered input.
func confirm(prompt string, r *bufio.Reader, out io.Writer) answer {
	if r == nil {
		return answerUnavailable
	}
	fmt.Fprint(out, prompt)
	line, err := r.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		// EOF before a single character: </dev/null, a closed pipe, a timer unit.
		fmt.Fprintln(out)
		return answerUnavailable
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return answerYes
	}
	return answerNo
}

// confirmProceed asks the standard "==> proceed?" question and turns the answer
// into the exit code the caller should return. A refusal is success; an
// unanswerable prompt is an error, because `gpk upgrade </dev/null` reporting
// exit 0 after upgrading nothing is worse than either.
func confirmProceed(st *styler, r *bufio.Reader, stdout, stderr io.Writer) (proceed bool, code int) {
	switch confirm(st.accent("==> proceed?")+" [y/N] ", r, stdout) {
	case answerYes:
		return true, ExitOK
	case answerNo:
		fmt.Fprintln(stderr, "cancelled")
		return false, ExitOK
	default:
		fmt.Fprintln(stderr, "error: no confirmation possible — stdin is closed or empty; pass --yes (or --noconfirm) to run unattended")
		return false, ExitErr
	}
}

// readSelection prints prompt and returns the raw line. ok is false on EOF.
func readSelection(prompt string, r *bufio.Reader, out io.Writer) (string, bool) {
	if r == nil {
		return "", false
	}
	fmt.Fprint(out, prompt)
	line, err := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		return "", false
	}
	return line, true
}

// parseSelection turns a yay-style selection into zero-based indices into a
// list of length n. It accepts space/comma separated numbers and inclusive
// ranges ("1 3", "2-4", "1,4"), plus "all"/"a" for everything. Out-of-range
// or malformed tokens produce an error.
func parseSelection(input string, n int) ([]int, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return nil, nil
	}
	if input == "all" || input == "a" || input == "*" {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out, nil
	}

	seen := make(map[int]bool)
	var out []int
	add := func(i int) error {
		if i < 1 || i > n {
			return fmt.Errorf("selection %d out of range (1-%d)", i, n)
		}
		if !seen[i-1] {
			seen[i-1] = true
			out = append(out, i-1)
		}
		return nil
	}

	fields := strings.FieldsFunc(input, func(r rune) bool { return r == ' ' || r == ',' })
	for _, f := range fields {
		if lo, hi, ok := strings.Cut(f, "-"); ok {
			a, err1 := strconv.Atoi(strings.TrimSpace(lo))
			b, err2 := strconv.Atoi(strings.TrimSpace(hi))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range %q", f)
			}
			if a > b {
				a, b = b, a
			}
			for i := a; i <= b; i++ {
				if err := add(i); err != nil {
					return nil, err
				}
			}
			continue
		}
		i, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("invalid selection %q", f)
		}
		if err := add(i); err != nil {
			return nil, err
		}
	}
	return out, nil
}
