#!/bin/bash
# run.sh
# Main dispatcher for the Linux kernel build environment on macOS.
# All logic is modularized into the 'scripts/' directory and sourced via common.env.

# Ensure execution stops immediately on error
set -e

# ──────────────────────────────────────────────────────────────────────────────
# 1. Setup Environment
# ──────────────────────────────────────────────────────────────────────────────
# Local definitions
HELPER_LENGTH=16

# Load environment variables, paths, and all function scripts
source "./common.env"

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
# Target Setup
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

# --------------------------------------------------
# Build Pipeline
# --------------------------------------------------
rootfs)
	create_disk_image
	;;
build)
	run_build "$@" # Pass all remaining arguments (jobs, targets)
	;;
clean)
	run_clean
	;;

# --------------------------------------------------
# Module Management
# --------------------------------------------------
module)
	run_module "$@"
	;;

# --------------------------------------------------
# Debugger Execution
# --------------------------------------------------
qemu)
	run_qemu "$@"
	;;

# --------------------------------------------------
# User Help Menu
# --------------------------------------------------
version)
	echo "Linux Kernel on macOS - Build & Management Script"
	printf "Version: ${PURPLE}${REPO_VERSION}${NC}\n"
	printf "Copyright (C) ${RELEASE_YEAR} ${AUTHOR_NAME}\n"
	printf "License: ${REPO_LICENSE}\n"
	printf "Repository: <${GREEN}${REPO_LINK}${NC}>\n"
	printf "Report: <${YELLOW}${REPO_LINK}/issues${NC}>\n"
	;;
help)
	echo "This is the kernel build and management helper script. Usage:"
	echo
	printf "  ${GREEN}Core Lifecycle${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "mount" "Mounts image (or creates) and ensures kernel repo exists."
	printf "    %-${HELPER_LENGTH}s %s\n" "unmount/exit" "Unmounts the sparse image."
	printf "    %-${HELPER_LENGTH}s %s\n" "(no arg)" "Alias for 'mount'."
	echo
	printf "  ${GREEN}Environment & Repo${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "doctor" "Checks dependencies and required headers."
	printf "    %-${HELPER_LENGTH}s %s\n" "status" "git status."
	printf "    %-${HELPER_LENGTH}s %s\n" "update" "git fetch + hard reset to origin/\$BASE_BRANCH."
	printf "    %-${HELPER_LENGTH}s %s\n" "reset" "git reset --hard origin/\$BASE_BRANCH (no fetch)."
	printf "    %-${HELPER_LENGTH}s %s\n" "reinit" "Deletes kernel source and clones fresh repo."
	printf "    %-${HELPER_LENGTH}s %s\n" "branch <name>" "Checkout branch/tag (creates new branch if missing)."
	printf "    %-${HELPER_LENGTH}s %s\n" "delete <name>" "Safely deletes a local branch."
	echo
	printf "  ${GREEN}Target Setup${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "arch <target>" "Sets target architecture (e.g., riscv, arm64)."
	printf "    %-${HELPER_LENGTH}s %s\n" "patch <file>" "Applies a patch file (e.g., patches/v6.18/0001.patch)."
	printf "    %-${HELPER_LENGTH}s %s\n" "config [cf]" "make defconfig or type (e.g., allnoconfig)."
	echo
	printf "  ${GREEN}Build Pipeline${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "rootfs" "Creates/packages minimal Debian Initramfs (debootstrap)."
	printf "    %-${HELPER_LENGTH}s %s\n" "build [jb] [tg]" "make -j[cores] target (default: Image dtbs modules)."
	printf "    %-${HELPER_LENGTH}s %s\n" "clean" "make distclean."
	echo
	printf "  ${GREEN}Module Management${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "module [km] [op]" "Manage kernel modules (build, insmod, rmmod, status)."
	echo
	printf "  ${GREEN}Debugger Execution${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "qemu [-d]" "Runs or debugs the built kernel in QEMU (using current ARCH)."
	echo
	printf "  ${GREEN}User Help Menu${NC}\n"
	printf "    %-${HELPER_LENGTH}s %s\n" "version" "Displays version and repository information."
	printf "    %-${HELPER_LENGTH}s %s\n" "help" "Displays this help menu."
	;;

# --------------------------------------------------
# Unknown Command
# --------------------------------------------------
*)
	echo "Usage: ./run.sh [command] [args]"
	echo "Run './run.sh help' for a list of commands."
	echo
	;;

esac
