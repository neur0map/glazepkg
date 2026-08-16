# Homebrew packaging

Two formulae, for two different jobs.

`gpk.rb` here is the **homebrew-core candidate**: it builds from the source
tarball, which is what core requires. It is not used by any release workflow;
it is the file to open a PR with against
[Homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core).

`.github/templates/gpk.rb` is the **tap formula**, generated on every release
into [neur0map/homebrew-tap](https://github.com/neur0map/homebrew-tap). It
installs the pre-built release binary. Core will not accept that, and the tap
keeps working for everyone already on it.

## Why the build tag

Core's [package acceptance policy](https://docs.brew.sh/Package-Acceptance-Policy)
says self-update behaviour must be disabled when that can be done without a
fragile patch. So the formula builds with `-tags noselfupdate`, which compiles
the updater out: `gpk update` returns an error and the binary contains no code
that can replace itself.

This is the same shape Syncthing uses. Its core formula passes `--no-upgrade`,
which sets a `noupgrade` build tag that swaps the upgrade implementation for a
stub exposing `DisabledByCompilation`.

The tag is opt-in. Every other channel (release downloads, the tap, AUR,
`go install`) builds without it and keeps a working `gpk update`. Those channels
are covered at runtime instead: `updater.Managed()` asks who owns the running
binary and refuses when the answer is a package manager.

## State of the submission

A branch is staged at `neur0map/homebrew-core`, branch `gpk`, one commit
`gpk 0.6.7 (new formula)` adding `Formula/g/gpk.rb`. The pull request itself is
not opened: homebrew-core's template asks the submitter to confirm they will
answer maintainer questions themselves, and to disclose any AI involvement, so
that is yours to fill in and send.

`brew audit --strict --online --new local/audit/gpk` passes clean against
v0.6.7 on brew 6.0.17. Note the flag: `--new-formula` was renamed to `--new`,
so older write-ups are out of date. Auditing a formula that is not in a tap
needs one:

```bash
brew tap-new local/audit --no-git
cp packaging/homebrew/gpk.rb "$(brew --repository local/audit)/Formula/"
brew audit --strict --online --new local/audit/gpk
```

Still to run, on a machine where `go` has a bottle:

```bash
HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source local/audit/gpk
brew test local/audit/gpk
```

Every command those two would run has been checked by hand against the
published v0.6.7 tarball: the declared sha256 matches, the build line
`std_go_args` expands to succeeds, completions generate for bash, zsh and fish,
and all four test assertions pass. What is unverified is Homebrew's own
plumbing around them.

The `url` and `sha256` must always point at a release that contains the build
tag, or `-tags noselfupdate` is a silent no-op, because Go accepts unknown tags
without complaint. The `test do` block exists to catch that: it asserts
`gpk update` exits non-zero. It caught exactly that mistake when the formula
was first pointed at v0.6.6.

Notability is met: 572 stars against the 225 core wants for a self-submission
by the repository owner, or 75 if somebody else opens it.

## Retiring the tap afterwards

Do not delete the tap formula on its own; that orphans everyone who installed
from it. Homebrew's mechanism is `tap_migrations.json` in the tap root, which
is what the argoproj and helm taps use:

```json
{ "gpk": "homebrew/core" }
```

Add that entry and remove `Formula/gpk.rb` from the tap in the same commit, so
`brew` can redirect the name. An unqualified `brew install gpk` prefers core
once it lands, while `brew install neur0map/tap/gpk` keeps resolving to the tap
until the formula is gone.

If you want the old path to warn before it stops working, keep the formula for
a grace period with:

```ruby
deprecate! date: "YYYY-MM-DD", because: "has been moved to homebrew/core", replacement_formula: "gpk"
```

There is no preset `because:` symbol for a move into core, so the string above
is the way to say it.
