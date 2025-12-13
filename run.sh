#!/bin/bash
# run.sh
# Script for managing Linux kernel build environment on macOS.
# Follows SOLID principles: single responsibility per function, open for extension, etc.
# Commands are handled in a case statement for clarity and maintainability.
# Assumptions:
# - Linux kernel repo: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git (mainline).
# - Base branch: master (can be extended in config if needed).
# - Headers setup: User must manually create $HOME/Documents/kernel-dev/linux/libraries and populate with necessary headers
# (e.g., elf.h from glibc elf/elf.h, byteswap.h custom, asm symlinks to kernel uapi/asm-generic).
# - Patches: Applied with 3-way merge; conflicts generate .rej files.
# - Build uses gmake with LLVM=1 for cross-compilation.
# - Default ARCH mapping: arm64 (for Apple Silicon) or x86_64 (for Intel).
# - Sparse image: Always at script's directory, mounted at /Volumes/kernel-dev.
# - Repo: Cloned into /Volumes/kernel-dev/linux if not present.
# - Config persisted in /Volumes/kernel-dev/config.env for TARGET_ARCH.
# - Mount required for most commands; use no args to mount, 'exit' to unmount.
# - Doctor checks basics; now includes headers verification and optional elf.h update from glibc.

set -e

source ./common.env # Loads PATH, HOSTCFLAGS, etc.

# ─────────────────────────────────────────────────────────────
# Global paths & defaults
# ─────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_FILE="${SCRIPT_DIR}/img.sparseimage"
VOLUME_NAME="kernel-dev"
MOUNT_POINT="/Volumes/${VOLUME_NAME}"
KERNEL_DIR="${MOUNT_POINT}/linux"
BASE_BRANCH="master"

# Default target arch = host arch
if [ "$(uname -m)" = "arm64" ]; then
	DEFAULT_ARCH="arm64"
else
	DEFAULT_ARCH="x86_64"
fi

# ─────────────────────────────────────────────────────────────
# Helper functions
# ─────────────────────────────────────────────────────────────
is_mounted() { mount | grep -q "${MOUNT_POINT}"; }

section() {
	echo "────────────────────────────────────────────"
	echo " $1"
	echo "────────────────────────────────────────────"
}

mount_image() {
	if [ ! -f "${IMAGE_FILE}" ]; then
		echo "Creating 20GB case-sensitive sparseimage..."
		hdiutil create -size 20g -fs 'Case-sensitive APFS' -type SPARSE -volname "${VOLUME_NAME}" "${IMAGE_FILE}"
	fi
	echo "Mounting kernel-dev volume..."
	hdiutil attach "${IMAGE_FILE}"
}

unmount_image() {
	if is_mounted; then
		echo "Unmounting ${MOUNT_POINT}..."
		hdiutil detach -force "${MOUNT_POINT}"
	fi
	echo "Current hdiutil state:"
	hdiutil info
}

ensure_mounted() {
	if ! is_mounted; then
		echo "Error: Volume not mounted. Run ./run.sh to mount first."
		exit 1
	fi
}

check_repo() {
	if [ ! -d "${KERNEL_DIR}/.git" ]; then
		echo "Cloning mainline Linux kernel into ${KERNEL_DIR}..."
		git clone https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git "${KERNEL_DIR}"
	fi
}

load_config() {
	[ -f "${MOUNT_POINT}/config.env" ] && source "${MOUNT_POINT}/config.env"
}

# ─────────────────────────────────────────────────────────────
# Doctor — fast & beautiful dependency check
# ─────────────────────────────────────────────────────────────
doctor() {
	echo "Running doctor — environment check"
	echo

	# 1. Homebrew packages (single fast call)
	section "Homebrew packages"
	local DEPENDENCIES=(llvm lld gnu-sed make libelf git)
	local installed=$(brew list --formulae)
	local missing=()

	for dep in "${DEPENDENCIES[@]}"; do
		[[ $installed == *"$dep"* ]] || missing+=("$dep")
	done

	if [ ${#missing[@]} -eq 0 ]; then
		echo "All required Homebrew packages are installed"
	else
		echo "Missing packages:"
		printf '   • %s\n' "${missing[@]}"
		echo
		echo "Fix with: brew install ${missing[*]}"
		echo
	fi

	# 2. Custom headers verification
	section "macOS host headers"
	local headers_dir="${HOME}/Documents/kernel-dev/linux/libraries"
	local ok_headers=1

	if [ ! -d "$headers_dir" ]; then
		echo "$headers_dir: MISSING (create it)"
		ok_headers=0
	else
		# Check byteswap.h (nothing changes)
		[ -f "$headers_dir/byteswap.h" ] && echo "byteswap.h: OK" || {
			echo "byteswap.h: MISSING"
			ok_headers=0
		}

		# Check endian.h (nothing changes)
		[ -f "$headers_dir/endian.h" ] && echo "endian.h: OK" || {
			echo "endian.h: MISSING"
			ok_headers=0
		}

		# Check elf.h
		[ -f "$headers_dir/elf.h" ] && echo "elf.h: OK" || {
			echo "elf.h: MISSING"
			ok_headers=0
		}

		# Check asm/ symlinks (linked to kernel uapi/asm-generic)
		local uapi_path="/Volumes/kernel-dev/linux/include/uapi/asm-generic"
		for h in bitsperlong.h int-ll64.h posix_types.h types.h; do
			local link="$headers_dir/asm/$h"
			if [ -L "$link" ] && [ "$(readlink "$link")" = "$uapi_path/$h" ]; then
				echo "asm/$h: OK (linked to $uapi_path/$h)"
			else
				echo "asm/$h: MISSING or incorrect link"
				ok_headers=0
			fi
		done
	fi

	# Optional: Update elf.h from glibc
	if [ $ok_headers -eq 0 ] || [ ! -f "$headers_dir/elf.h" ]; then
		echo "elf.h is missing or outdated. Update from glibc? (Y/n, default: latest)"
		read -r choice
		if [[ "${choice}" =~ ^[Nn]$ ]]; then
			echo "Skipped elf.h update"
		else
			# Default to latest (2.42 as of Dec 2025; can be dynamic)
			local glibc_ver="2.42"
			local glibc_url="https://raw.githubusercontent.com/bminor/glibc/glibc-${glibc_ver}/elf/elf.h"

			echo "Downloading elf.h header from glibc ${glibc_ver}..."
			mkdir -p "$headers_dir"
			if wget $glibc_url -P "$headers_dir"; then
				echo "Downloaded elf.h to $headers_dir successfully."
				ok_headers=1
			else
				echo "Failed to download elf.h from glibc."
			fi
		fi
	fi

	# Final verdict
	echo
	if [ ${#missing[@]} -eq 0 ] && [ $ok_headers -eq 1 ]; then
		echo "Doctor result: All good! Your macOS environment is 100% ready for Linux kernel builds."
	else
		echo "Doctor result: Issues found — fix above and re-run './run.sh doctor'"
		return 1
	fi
}

# ─────────────────────────────────────────────────────────────
# Main command dispatcher
# ─────────────────────────────────────────────────────────────
if [ $# -eq 0 ]; then
	mount_image
	check_repo
	exit 0
fi

case "$1" in
doctor) doctor ;;
status)
	ensure_mounted
	cd "$KERNEL_DIR"
	git status
	;;
reinit)
	ensure_mounted
	echo "This will delete the entire kernel tree and re-clone. Continue? (Y/n)"
	read -r ans
	[[ "$ans" == "n" || "$ans" == "N" ]] && echo "Aborted." && exit 0
	rm -rf "$KERNEL_DIR"
	check_repo
	;;
update)
	ensure_mounted
	cd "$KERNEL_DIR"
	git fetch origin
	git reset --hard "origin/$BASE_BRANCH"
	git clean -fd
	;;
branch)
	[[ -z "$2" ]] && {
		echo "Usage: $0 branch <name|tag>"
		exit 1
	}
	ensure_mounted
	cd "$KERNEL_DIR"

	case "$2" in
	v*) # v6.18, v6.17, etc.
		if git show-ref --tags --quiet "refs/tags/$2"; then
			echo "Official tag $2 found → checking out"
			git checkout "$2"
			exit 0
		fi
		;;
	esac

	# Existing local branch → just switch
	git show-ref --quiet "refs/heads/$2" && {
		git checkout "$2"
		exit 0
	}

	# Nothing found → create new branch tracking origin/master
	echo "Creating new branch $2 from origin/master"
	git checkout -b "$2" --track origin/master
	;;
arch)
	[[ -z "$2" ]] && echo "Usage: $0 arch <arm64|x86_64|...>" && exit 1
	ensure_mounted
	echo "export TARGET_ARCH=\"$2\"" >"$MOUNT_POINT/config.env"
	echo "Target architecture set to: $2"
	;;
patch)
	[[ -z "$2" ]] && {
		echo "Usage: $0 patch <file>"
		exit 1
	}
	ensure_mounted
	cd "$KERNEL_DIR"

	echo "Applying $2 ..."
	if git am --3way "$SCRIPT_DIR/$2"; then
		echo "Applied cleanly"
	else
		echo "Conflict! Applying with .rej files..."
		git am --abort 2>/dev/null
		git apply --reject "$SCRIPT_DIR/$2" && echo "Check .rej files!" || {
			echo "Patch failed completely"
			exit 1
		}
	fi
	;;
config)
	ensure_mounted
	load_config
	arch="${TARGET_ARCH:-$DEFAULT_ARCH}"
	cd "$KERNEL_DIR"
	gmake ARCH="$arch" LLVM=1 "${2:-defconfig}"
	;;
build)
	ensure_mounted
	load_config
	arch="${TARGET_ARCH:-$DEFAULT_ARCH}"
	jobs="${2:-$(sysctl -n hw.ncpu)}"

	# shift past "build" and optional job count
	shift
	if [ -n "$1" ] && [ "$1" -eq "$1" ] 2>/dev/null; then
		jobs="$1"
		shift
	fi

	# if anything left → user targets, else default
	targets="${*:-Image dtbs modules}"

	cd "$KERNEL_DIR"
	echo "Building ARCH=$arch | $jobs jobs | $targets"
	gmake ARCH="$arch" LLVM=1 -j"$jobs" $targets
	;;
clean)
	ensure_mounted
	load_config
	arch="${TARGET_ARCH:-$DEFAULT_ARCH}"
	cd "$KERNEL_DIR"
	gmake ARCH="$arch" LLVM=1 distclean
	;;
reset)
	ensure_mounted
	cd "$KERNEL_DIR"
	echo "This will discard all local changes and reset to origin/$BASE_BRANCH. Continue? (Y/n)"
	read -r ans
	[[ "$ans" == "n" || "$ans" == "N" ]] && echo "Aborted." && exit 0
	git reset --hard "origin/$BASE_BRANCH"
	git clean -fd
	;;
delete)
	[[ -z "$2" ]] && {
		echo "Usage: $0 delete <branch>"
		exit 1
	}

	ensure_mounted
	cd "$KERNEL_DIR"

	BRANCH="$2"

	# Safety: refuse to delete master or main
	if [[ "$BRANCH" == "master" || "$BRANCH" == "main" ]]; then
		echo "Error: Cowardly refusing to delete '$BRANCH'"
		exit 1
	fi

	# Must not be on the branch you're deleting
	CURRENT=$(git branch --show-current)
	if [[ "$CURRENT" == "$BRANCH" ]]; then
		echo "Error: You are currently on '$BRANCH'. Switch away first."
		exit 1
	fi

	# Delete only local branch
	if git show-ref --quiet "refs/heads/$BRANCH"; then
		git branch -D "$BRANCH"
		echo "Deleted local branch: $BRANCH"
	else
		echo "Branch '$BRANCH' does not exist locally"
	fi
	;;
exit)
	unmount_image
	;;
help | *)
	cat <<EOF
Usage: ./run.sh [command] [args]

No argument → mount (or create) img.sparseimage + ensure kernel repo

Commands:
  doctor                    Fast dependency & environment check
  status                    git status
  reinit                    Delete & re-clone kernel repo
  update                    Fetch + hard reset to upstream master
  branch <name>             Checkout (create if missing)
  arch <target>             Set target arch (arm64, x86_64, etc.)
  patch <file>              Apply a patch file (with 3-way merge)
  config [type]             make defconfig (or allnoconfig, etc.)
  build [jobs] [targets]    Build (default: all cores, Image dtbs modules)
  clean                     make distclean
  reset                     git reset --hard origin/master
  delete <branch>           Safe delete a local branch
  exit                      Unmount volume
  help                      This message

Enjoy building Linux on macOS!
EOF
	[[ "$1" != "help" ]] && exit 1
	;;
esac
