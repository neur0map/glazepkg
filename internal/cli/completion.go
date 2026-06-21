package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["completion"] = runCompletion
	subcommands["completions"] = runCompletions
}

// runCompletions is the fast helper the shell scripts call on every <Tab>. It
// only reads cached state — never a live scan — so completion stays instant.
//
//	commands   subcommand names
//	managers   manager names (and friendly aliases)
//	installed  installed package names (from the scan cache)
//	held       held package names
func runCompletions(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	kind := ""
	if len(args) > 0 {
		kind = args[0]
	}
	switch kind {
	case "commands":
		for _, n := range SubcommandNames() {
			if n == "completions" {
				continue
			}
			fmt.Fprintln(stdout, n)
		}
	case "managers":
		for _, m := range mgrs {
			fmt.Fprintln(stdout, m.Name())
		}
		for alias := range managerAliases {
			fmt.Fprintln(stdout, alias)
		}
	case "installed":
		seen := make(map[string]bool)
		var names []string
		for _, p := range manager.LoadScanCache() {
			if !seen[p.Name] {
				seen[p.Name] = true
				names = append(names, p.Name)
			}
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintln(stdout, n)
		}
	case "held":
		for _, h := range snapshot.LoadHolds() {
			fmt.Fprintln(stdout, h.Name)
		}
	}
	return ExitOK
}

func runCompletion(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	}
	script, ok := completionScripts[shell]
	if !ok {
		fmt.Fprintln(stderr, "usage: gpk completion <bash|zsh|fish>")
		return ExitErr
	}
	fmt.Fprint(stdout, script)
	return ExitOK
}

var completionScripts = map[string]string{
	"bash": `_gpk() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    local cmd="${COMP_WORDS[1]}"
    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=( $(compgen -W "$(gpk completions commands)" -- "$cur") )
        return
    fi
    case "$prev" in
        -m|--manager)
            COMPREPLY=( $(compgen -W "$(gpk completions managers)" -- "$cur") )
            return ;;
    esac
    case "$cmd" in
        remove|upgrade|info|source-of|why|versions|downgrade|hold)
            COMPREPLY=( $(compgen -W "$(gpk completions installed)" -- "$cur") ) ;;
        unhold)
            COMPREPLY=( $(compgen -W "$(gpk completions held)" -- "$cur") ) ;;
    esac
}
complete -F _gpk gpk
`,
	"zsh": `#compdef gpk
_gpk() {
    local cmd="${words[2]}"
    if (( CURRENT == 2 )); then
        compadd -- ${(f)"$(gpk completions commands)"}
        return
    fi
    case "${words[CURRENT-1]}" in
        -m|--manager) compadd -- ${(f)"$(gpk completions managers)"}; return ;;
    esac
    case "$cmd" in
        remove|upgrade|info|source-of|why|versions|downgrade|hold) compadd -- ${(f)"$(gpk completions installed)"} ;;
        unhold) compadd -- ${(f)"$(gpk completions held)"} ;;
    esac
}
compdef _gpk gpk
`,
	"fish": `complete -c gpk -f
complete -c gpk -n "__fish_use_subcommand" -a "(gpk completions commands)"
complete -c gpk -n "__fish_seen_subcommand_from remove upgrade info source-of why versions downgrade hold" -a "(gpk completions installed)"
complete -c gpk -n "__fish_seen_subcommand_from unhold" -a "(gpk completions held)"
complete -c gpk -l manager -x -a "(gpk completions managers)"
complete -c gpk -s m -x -a "(gpk completions managers)"
`,
}
