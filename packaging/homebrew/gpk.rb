class Gpk < Formula
  desc "TUI and CLI that unifies every package manager on the system"
  homepage "https://github.com/neur0map/glazepkg"
  url "https://github.com/neur0map/glazepkg/archive/refs/tags/v0.6.7.tar.gz"
  sha256 "9dcdd0b102d8f5ae167c8215c9f730f85c0a712a7bb512d78fabea47f6616b14"
  license "GPL-3.0-or-later"
  head "https://github.com/neur0map/glazepkg.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-X main.version=v#{version}", tags: "noselfupdate"), "./cmd/gpk"
    generate_completions_from_executable(bin/"gpk", "completion")
  end

  test do
    require "json"

    assert_match "gpk v#{version}", shell_output("#{bin}/gpk --version")

    # Detect a package manager that really is on this machine, and report it.
    managers = JSON.parse(shell_output("#{bin}/gpk managers --json --quiet"))
    assert_equal 1, managers["schema"]
    brew_entry = managers["data"].find { |entry| entry["name"] == "brew" }
    refute_nil brew_entry, "gpk did not detect Homebrew"
    assert_equal true, brew_entry["available"]

    # Scan through that manager and emit the documented envelope.
    listed = JSON.parse(shell_output("#{bin}/gpk list --json --manager brew --quiet"))
    assert_equal 1, listed["schema"]
    assert_kind_of Array, listed["data"]

    # Resolve an installed package back to the manager that owns it.
    assert_match "brew", shell_output("#{bin}/gpk source-of gpk")

    # Exit 2 is the documented "meaningful no", not a failure.
    shell_output("#{bin}/gpk installed gpk-does-not-exist --quiet", 2)

    # Snapshot, then diff it against the live system.
    system bin/"gpk", "snapshot", "--manager", "brew", "--quiet"
    assert_match "1 snapshot", shell_output("#{bin}/gpk snapshot list")
    assert_match "no changes", shell_output("#{bin}/gpk snapshot diff --manager brew --quiet")

    # This build cannot replace its own binary.
    assert_match(/upgrade/i, shell_output("#{bin}/gpk update 2>&1", 1))
  end
end
