# Formula for Homebrew installation of kai
# This file should be placed in a homebrew-kai tap repository
# OR use directly: brew install tinhvqbk/kai/kai

class Kai < Formula
  desc "Knowledge AI — your digital twin. CLI-based AI agent with persistent brain."
  homepage "https://github.com/tinhvqbk/kai"
  license "MIT"

  # Update these when releasing new versions
  version "1.0.0"

  on_macos do
    on_intel do
      url "https://github.com/tinhvqbk/kai/releases/download/v#{version}/kai-darwin-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_INTEL"
    end

    on_arm do
      url "https://github.com/tinhvqbk/kai/releases/download/v#{version}/kai-darwin-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_ARM64"
    end
  end

  on_linux do
    url "https://github.com/tinhvqbk/kai/releases/download/v#{version}/kai-linux-amd64"
    sha256 "REPLACE_WITH_ACTUAL_SHA256_LINUX"
  end

  def install
    bin.install "kai-#{os.kernel_name.downcase}-#{hardware.arch_name}" => "kai"
  end

  test do
    system "#{bin}/kai", "version"
  end
end
