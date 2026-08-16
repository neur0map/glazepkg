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

## Before submitting

The formula's `url` and `sha256` must point at a release that contains the
build tag, or `-tags noselfupdate` is a silent no-op — Go accepts unknown tags
without complaint. The `test do` block catches exactly that: it asserts
`gpk update` exits non-zero.

Run the audit before opening the PR:

```bash
brew audit --strict --online --new-formula gpk
brew install --build-from-source ./packaging/homebrew/gpk.rb
brew test gpk
```

Notability was already met at submission time: 572 stars against the 225 that
core wants for a self-submission by the repository owner (75 if someone else
opens the PR).

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
