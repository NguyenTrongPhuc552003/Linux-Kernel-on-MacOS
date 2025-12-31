# typed: false
# frozen_string_literal: true

# ELMOS - Embedded Linux on MacOS
# Homebrew formula for installing elmos CLI tool
#
# Installation:
#   brew tap NguyenTrongPhuc552003/elmos
#   brew install elmos
#
# Development install (latest from git):
#   brew install --HEAD elmos

class Elmos < Formula
  desc "Embedded Linux on MacOS - Native kernel build tools"
  homepage "https://github.com/NguyenTrongPhuc552003/elmos"
  license "MIT"

  head "https://github.com/NguyenTrongPhuc552003/elmos.git", branch: "main"
  url "https://github.com/NguyenTrongPhuc552003/elmos/archive/refs/tags/v3.0.0.tar.gz"
  sha256 "8222933d906bdb973dbe4ffabb4a7f7318530a4372e741bbf986b359bc0468fd"
  version "3.0.0"

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
    # Get version from git
    version_str = Utils.safe_popen_read("git", "describe", "--tags", "--always", "--dirty").strip
    commit = Utils.safe_popen_read("git", "rev-parse", "--short", "HEAD").strip
    build_date = Time.now.utc.iso8601

    ldflags = %W[
      -s -w
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.Version=#{version_str}
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.Commit=#{commit}
      -X github.com/NguyenTrongPhuc552003/elmos/pkg/version.BuildDate=#{build_date}
    ]

    system "go", "build", *std_go_args(ldflags:)

    # Generate and install shell completions
    generate_completions_from_executable(bin/"elmos", "completion")

    # Install supporting files
    pkgshare.install "libraries" if File.directory?("libraries")
    pkgshare.install "patches" if File.directory?("patches")
  end

  def caveats
    <<~EOS
      ELMOS has been installed!

      Quick start:
        elmos doctor              # Check dependencies
        elmos ui                  # Launch interactive TUI
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
    assert_match "ELMOS", shell_output("#{bin}/elmos --help")
    assert_match "Version", shell_output("#{bin}/elmos version")
  end
end
