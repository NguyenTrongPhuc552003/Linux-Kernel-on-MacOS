#!/bin/bash
# run.sh
# Main dispatcher for the Linux kernel build environment on macOS.
# All logic is modularized into the 'scripts/' directory and sourced via common.env.

# Ensure execution stops immediately on error
set -e

# ──────────────────────────────────────────────────────────────────────────────
# 1. Setup Environment
# ──────────────────────────────────────────────────────────────────────────────
# Define the script directory relative to this file
export SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load environment variables, paths, and all function scripts
source "${SCRIPT_DIR}/common.env"

# ──────────────────────────────────────────────────────────────────────────────
# 2. Command Dispatcher
# ──────────────────────────────────────────────────────────────────────────────
COMMAND="$1"
shift || true # Remove command from arguments, or do nothing if no arguments

case "$COMMAND" in
# --------------------------------------------------
# Core Lifecycle
# --------------------------------------------------
mount)
	mount_image
	check_repo # Ensure kernel source is ready after mounting
	;;
unmount | exit)
	unmount_image
	;;
# No argument: Default action is to mount and check repo
"")
	mount_image
	check_repo
	;;

# --------------------------------------------------
# Doctor / Repo / Branch
# --------------------------------------------------
doctor)
	run_doctor
	;;
status)
	repo_status
	;;
reinit)
	repo_reinit
	;;
update)
	repo_update
	;;
reset)
	repo_reset
	;;
branch)
	git_branch "$1"
	;;
delete)
	delete_branch "$1"
	;;

# --------------------------------------------------
# Build
# --------------------------------------------------
arch)
	set_arch "$1"
	;;
patch)
	apply_patch "$1"
	;;
config)
	run_config "$1"
	;;
rootfs)
	package_initramfs
	;;
build)
	run_build "$@" # Pass all remaining arguments (jobs, targets)
	;;
clean)
	run_clean
	;;
qemu)
	# Check for a second argument
	if [ "$2" = "-d" ]; then
		run_qemu_gdb
	else
		run_qemu
	fi
	;;

# --------------------------------------------------
# Help / Error
# --------------------------------------------------
help | *)
	echo "Usage: ./run.sh [command] [args]"
	echo "Run './run.sh help' for a list of commands."
	echo
	echo "Available commands (source functions from scripts/):"
	echo
	printf "  ${GREEN}Core Lifecycle${NC}\n"
	printf "    %-15s %s\n" "mount" "Mounts image (or creates) and ensures kernel repo exists."
	printf "    %-15s %s\n" "unmount/exit" "Unmounts the sparse image."
	printf "    %-15s %s\n" "(no arg)" "Alias for 'mount'."
	echo
	printf "  ${GREEN}Environment & Repo${NC}\n"
	printf "    %-15s %s\n" "doctor" "Checks dependencies and required headers."
	printf "    %-15s %s\n" "status" "git status."
	printf "    %-15s %s\n" "update" "git fetch + hard reset to origin/\$BASE_BRANCH."
	printf "    %-15s %s\n" "reset" "git reset --hard origin/\$BASE_BRANCH (no fetch)."
	printf "    %-15s %s\n" "reinit" "Deletes kernel source and clones fresh repo."
	printf "    %-15s %s\n" "branch <name>" "Checkout branch/tag (creates new branch if missing)."
	printf "    %-15s %s\n" "delete <name>" "Safely deletes a local branch."
	echo
	printf "  ${GREEN}Build Pipeline${NC}\n"
	printf "    %-15s %s\n" "arch <target>" "Sets target architecture (e.g., arm64, x86_64)."
	printf "    %-15s %s\n" "patch <file>" "Applies a patch file (e.g., patches/v6.18/0001.patch)."
	printf "    %-15s %s\n" "config [type]" "make defconfig or type (e.g., allnoconfig)."
	printf "    %-15s %s\n" "rootfs" "Creates/packages minimal Debian Initramfs (debootstrap)."
	printf "    %-15s %s\n" "build [jb] [tg]" "make -j[cores] target (default: Image dtbs modules)."
	printf "    %-15s %s\n" "clean" "make distclean."
	printf "    %-15s %s\n" "qemu [-d]" "Runs or debugs the built kernel in QEMU (using current ARCH)."
	;;
esac
