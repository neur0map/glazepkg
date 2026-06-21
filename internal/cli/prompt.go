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

// confirm prints prompt and returns true only for y/yes. A shared reader is
// used so a later read in the same flow doesn't lose buffered input.
func confirm(prompt string, r *bufio.Reader, out io.Writer) bool {
	if r == nil {
		return false
	}
	fmt.Fprint(out, prompt)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
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
