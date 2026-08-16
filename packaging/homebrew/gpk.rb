class Gpk < Formula
  desc "TUI and CLI that unifies every package manager on the system"
  homepage "https://github.com/neur0map/glazepkg"
  url "https://github.com/neur0map/glazepkg/archive/refs/tags/v0.6.7.tar.gz"
  sha256 "9dcdd0b102d8f5ae167c8215c9f730f85c0a712a7bb512d78fabea47f6616b14"
  license "GPL-3.0-or-later"
  head "https://github.com/neur0map/glazepkg.git", branch: "main"

  livecheck do
    url :stable
    strategy :github_latest
  end

  depends_on "go" => :build

  def install
    # noselfupdate compiles out the self-updater, so `gpk update` refuses
    # instead of replacing the binary Homebrew owns.
    system "go", "build", *std_go_args(ldflags: "-X main.version=v#{version}", tags: "noselfupdate"), "./cmd/gpk"
    generate_completions_from_executable(bin/"gpk", "completion")
  end

  test do
    assert_match "gpk v#{version}", shell_output("#{bin}/gpk --version")
    assert_match "PACMAN / YAY FLAGS", shell_output("#{bin}/gpk --help")
    assert_match "_gpk", shell_output("#{bin}/gpk completion bash")

    # This build must never replace its own binary.
    assert_match(/upgrade/i, shell_output("#{bin}/gpk update 2>&1", 1))
  end
end
