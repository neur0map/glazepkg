# GlazePKG Roadmap

gpk began as a way to see everything installed on your machine in one place. The next step is to let you act on it: install, update, remove, and tidy up, with one tool, across every package manager you use.

This is the direction I want to take gpk. It is not fixed. Details and order will shift as I go, and I'm happy to hear suggestions.

## What gpk is

gpk is a helper that sits on top of the package tools you already have: brew, pacman, apt, nix, npm, and the rest. It gives them one shared set of commands, so you don't have to remember how each one spells "install" or asks "what's out of date."

The closest comparison is yay. yay gives Arch users one way to handle both the official packages and the community ones (the AUR). gpk takes that idea wider: one front end for every package tool on your system, whatever the platform.

## Two ways to use it

There is the full-screen view (the TUI) and there are typed commands (the CLI). The full-screen view is how most people use gpk today, and it is the more finished side. The typed commands are the part I'm building up now, so you can drive gpk straight from the terminal the way you would use yay, without opening the view at all. Both reach the same features. Stay in the view you like, or live in the command line. Neither one is going away.

## What gpk is not

- It is not a replacement for brew, pacman, apt, nix, or anything else. It drives those tools, it does not stand in for them. Remove gpk tomorrow and your software and your tools are exactly where you left them.
- It is not a new place to get software. It uses the sources you already trust.
- It is not trying to own your system. It is the one place you reach for that knows how to talk to all the others.

## Where it runs

Arch and other Linux, macOS, and Windows. The same commands work everywhere. gpk checks which tools you actually have and talks to those.

## What works today

- See everything installed across every tool, as a table or plain text (`gpk list`).
- Look up a single package: version, source, size, description (`gpk info`).
- Search for and install software across the tools you have (`gpk install`).
- Upgrade a package, or several at once.
- Remove a package, with an option on some tools to take its unused dependencies with it.
- Select many packages at once and act on them together (in the full-screen view).
- Save a snapshot of what is installed and compare it later to see what changed.
- Export your package list for backup or moving to a new machine.
- Several built-in color themes, with the option to add your own.
- Update gpk itself (`gpk update`).

## What needs work first

These are the gaps I most want to close, roughly in order.

1. **One command to update everything.** Today you upgrade packages one at a time. There is no single "bring it all up to date" yet. This is the biggest missing piece for everyday use.

2. **A smarter install.**
   - When a name exists in more than one tool, show a numbered list and let you pick, instead of stopping and asking you to add a flag.
   - When you mistype, suggest the closest match, the way yay does ("did you mean ...").
   - Before anything runs, show what will actually change: what gets added, what comes along as a dependency, and how much it downloads.

3. **Cleaning up.** A command to remove leftover dependencies nothing needs anymore, and one to clear out old download caches.

4. **Going back.** Downgrade a package to an earlier version, hold a package so it stays put during updates, and undo the last thing gpk did.

5. **Being dependable.**
   - Read each tool's output the same way no matter the system language.
   - Make search quick by keeping a local index, instead of waiting on each tool every time.
   - Keep a short history of what gpk changed, so you can look back and reverse it.

6. **Proper nix support.** On NixOS, installing should add the package to your configuration and rebuild, which is how NixOS is meant to work, with a quick way to just try something without keeping it. The current method uses an older command that does not fit flake-based systems. The compatibility table is also out of date for nix and needs to match the code.

## Familiar commands

Most people who use pacman or yay know the short flags by heart. `-S` to install, `-R` to remove, `-Syu` to update everything. Those flags are not owned by anyone. yay took them from pacman, paru took them too. gpk can offer the same shortcuts for people who want them, next to the plain words anyone can read at a glance.

So you will be able to write it either way:

| What you want | Plain words | Short flags |
|---|---|---|
| Install a package | `gpk install foo` | `gpk -S foo` |
| Remove a package | `gpk remove foo` | `gpk -R foo` |
| Update everything | `gpk upgrade` | `gpk -Syu` |
| Search | `gpk search foo` | `gpk -Ss foo` |
| List installed | `gpk list` | `gpk -Q` |

brew users keep the word style they already use. Arch users keep their muscle memory. Same tool either way.

## Where this is heading

Further out, once the basics above are solid:

- A first run that looks at your system, finds the tools you have, and shows you what it found so you can fix anything it got wrong.
- The "did you mean" suggestions and quick search from the list above, made the normal experience rather than extras.
- A queue, so you can line up several actions instead of waiting for each one to finish.
- Choosing a specific version at install time, not always the newest.
- A way for other people to add support for their own package tools, once the internal layout settles enough to keep it stable.
- Maybe, much later, building AUR packages directly instead of going through yay or paru. That is a large job on its own, so for now gpk uses whichever AUR helper you already have.
- Tracking whether something was installed for the whole system or just your user, which changes how it should be removed.

## How the vision has changed

gpk started as something you look at: one screen showing everything every tool had installed. People liked that, then asked to actually do things from the same screen. So it is turning into something you use, not just read. The aim stays the same. Not to become the one tool that owns your packages, but to be the one place you go that knows how to work with all of them.

## Themes

Several color themes ship with gpk, and `t` cycles through them live. Your choice is saved to `~/.local/share/glazepkg/theme.json`. Custom themes go in `~/.local/share/glazepkg/themes/` as a small file with the same color values.

## What gpk will not do

- Replace your package managers. It works with them.
- Turn into a server or enterprise management tool. It is a personal tool.
- Host its own software repository.

## Still open

- Every tool writes versions differently (brew `1.14.1`, pacman `1.14.1-1`, apt `1.14.1-2ubuntu1`, pip `1.14.1.post1`). Comparing them across tools is hard. gpk does its best and falls back to a plain text compare.
- Some tools can install software for the whole system or just for your user. gpk does not track that split yet, and it matters when removing.

## Contributing

Each piece above can be picked up on its own. See [CONTRIBUTING.md](CONTRIBUTING.md) for how the code is laid out. Open an issue or start a discussion if you want to take something on.
