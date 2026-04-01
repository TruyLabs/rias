class Kai < Formula
  desc "Your digital twin — powered by knowledge, not prompts"
  homepage "https://github.com/norenis/kai"
  license "MIT"
  version "$VERSION_NUM"

  on_macos do
    on_arm do
      url "https://github.com/norenis/kai/releases/download/v#{version}/kai-darwin-arm64.tar.gz"
      sha256 "$DARWIN_ARM64_SHA256"
    end
    on_intel do
      url "https://github.com/norenis/kai/releases/download/v#{version}/kai-darwin-amd64.tar.gz"
      sha256 "$DARWIN_AMD64_SHA256"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/norenis/kai/releases/download/v#{version}/kai-linux-amd64.tar.gz"
      sha256 "$LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install "kai"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kai version")
  end
end
