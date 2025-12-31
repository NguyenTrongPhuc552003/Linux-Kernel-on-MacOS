# typed: false
# frozen_string_literal: true

# ELMOS - Embedded Linux on MacOS
# Homebrew formula for installing elmos CLI tool
#
# Usage:
#   brew tap NguyenTrongPhuc552003/elmos
#   brew install elmos
#
# Or directly:
#   brew install NguyenTrongPhuc552003/elmos/elmos

class Elmos < Formula
  desc "Embedded Linux on MacOS - Native kernel build tools"
  homepage "https://github.com/NguyenTrongPhuc552003/elmos"
  url "https://github.com/NguyenTrongPhuc552003/elmos/archive/refs/tags/v3.0.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"
  license "MIT"
  head "https://github.com/NguyenTrongPhuc552003/elmos.git", branch: "main"

  depends_on "go" => :build
  depends_on "go-task" => :build

  # Runtime dependencies
  depends_on "llvm"
  depends_on "lld"
  depends_on "gnu-sed"
  depends_on "make"
  depends_on "libelf"
  depends_on "qemu"
  depends_on "e2fsprogs"
  depends_on "coreutils"
  depends_on "fakeroot"

  def install
    # Build with version info
    ldflags = %W[
      -s -w
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.Version=#{version}
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.Commit=#{tap.user}
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.BuildDate=#{time.iso8601}
    ]

    system "go", "build", *std_go_args(ldflags:)

    # Generate and install shell completions
    generate_completions_from_executable(bin/"elmos", "completion")

    # Install supporting files
    pkgshare.install "libraries"
    pkgshare.install "patches"
  end

  def caveats
    <<~EOS
      ELMOS has been installed!

      Quick start:
        elmos doctor              # Check dependencies
        elmos init                # Initialize workspace
        elmos config set arch arm64
        elmos kernel config
        elmos build
        elmos qemu run

      Required Homebrew tap for cross-toolchains:
        brew tap messense/macos-cross-toolchains

      Shell completions have been installed for bash, zsh, and fish.
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/elmos version")
    assert_match "ELMOS", shell_output("#{bin}/elmos --help")
  end
end
