#!/bin/bash
# Python/Services/EnvironmentService.sh
# Central configuration for the hybrid Python/Bash environment

# 1. Path Definitions
# -------------------
# Robustly find where THIS file is, regardless of where it is sourced from
export PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Standard Directories
export TOOLS_DIR="${PROJECT_ROOT}/tools"
export MODULES_DIR="${PROJECT_ROOT}/modules"
export LIBRARIES_DIR="${PROJECT_ROOT}/libraries"
export PATCH_DIR="${PROJECT_ROOT}/patches"

# State & Config
export STATE_DIR="${PROJECT_ROOT}/var/state"
export CONFIG_FILE="${STATE_DIR}/build.cfg"
export IMAGE_FILE="${STATE_DIR}/img.sparseimage"
export MOD_CONFIG="${MODULES_DIR}/module.cfg"

# 2. Load Persisted Config (if available)
if [ -f "$CONFIG_FILE" ]; then
	source "$CONFIG_FILE"
fi

# 3. CRITICAL: Inject Homebrew Tools into PATH
# -------------------------------------------
# macOS default tools (sed, awk, make) are often too old or BSD-variant.
# We prioritize Homebrew GNU tools.

if command -v brew >/dev/null 2>&1; then
	# GNU sed (Essential: BSD sed breaks kernel scripts)
	if [ -d "$(brew --prefix gnu-sed)/libexec/gnubin" ]; then
		export PATH="$(brew --prefix gnu-sed)/libexec/gnubin:$PATH"
	fi

	# LLVM / Clang (Ensure we use brew version, not Apple Clang)
	if [ -d "$(brew --prefix llvm)/bin" ]; then
		export PATH="$(brew --prefix llvm)/bin:$PATH"
	fi

	# LLD Linker
	if [ -d "$(brew --prefix lld)/bin" ]; then
		export PATH="$(brew --prefix lld)/bin:$PATH"
	fi

	# Coreutils (cp, mv, etc.)
	if [ -d "$(brew --prefix coreutils)/libexec/gnubin" ]; then
		export PATH="$(brew --prefix coreutils)/libexec/gnubin:$PATH"
	fi

	# E2fsprogs (mkfs.ext4)
	if [ -d "$(brew --prefix e2fsprogs)/sbin" ]; then
		export PATH="$(brew --prefix e2fsprogs)/sbin:$PATH"
	fi

	# Make (Ensure GNU Make >= 4.0)
	if [ -d "$(brew --prefix make)/libexec/gnubin" ]; then
		export PATH="$(brew --prefix make)/libexec/gnubin:$PATH"
	fi
fi

# 4. Helper Functions
# -------------------
ensure_mounted() {
    # Check if mount point exists and is not empty
    if [ ! -d "$MOUNT_POINT" ] || [ -z "$(ls -A "$MOUNT_POINT" 2>/dev/null)" ]; then
        echo "  [AUTO] Mounting workspace..."
        "$(dirname "${BASH_SOURCE[0]}")/ImageService.sh" mount
        if [ $? -ne 0 ]; then
             echo "  [ERROR] Failed to mount image."
             exit 1
        fi
    fi
}

# 5. Kernel & Build Constants
# ---------------------------
export REPO_VERSION="3.0.0"
export VOLUME_NAME="kernel-dev"
export MOUNT_POINT="/Volumes/${VOLUME_NAME}"
export KERNEL_DIR="${MOUNT_POINT}/linux"
export ROOTFS_DIR="${MOUNT_POINT}/rootfs"
export DISK_IMAGE="${MOUNT_POINT}/disk.img"
export DISK_SIZE="5G"

# 6. Compiler Flags & Includes
# ----------------------------
if command -v brew >/dev/null 2>&1; then
	export LIBELF_INCLUDE="$(brew --prefix libelf 2>/dev/null)/include"
else
	export LIBELF_INCLUDE="/usr/local/include"
fi

# macOS-specific flags for host tools (fix missing headers)
export NATIVE_FLAGS="-D_UUID_T -D__GETHOSTUUID_H -D_DARWIN_C_SOURCE -I${LIBRARIES_DIR} -I${LIBELF_INCLUDE}"
export HOSTCFLAGS="${NATIVE_FLAGS}"
export HOSTCXXFLAGS="${NATIVE_FLAGS}"

# Ensure these flags are passed to the kernel build system
export HOST_EXTRACFLAGS="${NATIVE_FLAGS}"
