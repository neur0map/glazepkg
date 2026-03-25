class Gpk < Formula
  desc "TUI dashboard that unifies 34 package managers into one searchable view"
  homepage "https://github.com/neur0map/glazepkg"
  version "${VER}"
  license "GPL-3.0-or-later"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/neur0map/glazepkg/releases/download/${TAG}/gpk-darwin-arm64"
      sha256 "${SHA_DARWIN_ARM64}"
    else
      url "https://github.com/neur0map/glazepkg/releases/download/${TAG}/gpk-darwin-amd64"
      sha256 "${SHA_DARWIN_AMD64}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/neur0map/glazepkg/releases/download/${TAG}/gpk-linux-arm64"
      sha256 "${SHA_LINUX_ARM64}"
    else
      url "https://github.com/neur0map/glazepkg/releases/download/${TAG}/gpk-linux-amd64"
      sha256 "${SHA_LINUX_AMD64}"
    end
  end

  def install
    bin.install Dir["gpk-*"].first => "gpk"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gpk --version")
  end
end
