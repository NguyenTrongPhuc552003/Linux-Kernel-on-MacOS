#!/bin/bash
# Python/Services/EnvironmentService.sh
# Central configuration for the hybrid Python/Bash environment

# 1. Path Definitions
# -------------------
# Robustly find where THIS file is, regardless of where it is sourced from
THIS_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Project Root is two levels up from Python/Services/
export PROJECT_ROOT="$(cd "${THIS_SCRIPT_DIR}/../.." && pwd)"

# Standard Directories
export TOOLS_DIR="${PROJECT_ROOT}/tools"
export MODULES_DIR="${PROJECT_ROOT}/modules"
export LIBRARIES_DIR="${PROJECT_ROOT}/libraries"
export PATCH_DIR="${PROJECT_ROOT}/patches"

# State & Config (Moved to var/state)
export STATE_DIR="${PROJECT_ROOT}/var/state"
export CONFIG_FILE="${STATE_DIR}/build.cfg"
export IMAGE_FILE="${STATE_DIR}/img.sparseimage"
export MOD_CONFIG="${MODULES_DIR}/module.cfg"

# 2. Kernel & Build Constants
# ---------------------------
export REPO_VERSION="2.0.0"
export VOLUME_NAME="kernel-dev"
export MOUNT_POINT="/Volumes/${VOLUME_NAME}"
export KERNEL_DIR="${MOUNT_POINT}/linux"
export ROOTFS_DIR="${MOUNT_POINT}/rootfs"
export DISK_IMAGE="${MOUNT_POINT}/disk.img"
export DISK_SIZE="5G"

# 3. Compiler & Flags
# -------------------
export MACOS_HEADERS="${LIBRARIES_DIR}"
# Handle brew errors gracefully if libelf isn't found
if command -v brew >/dev/null 2>&1; then
	export LIBELF_INCLUDE="$(brew --prefix libelf 2>/dev/null)/include"
else
	export LIBELF_INCLUDE="/usr/local/include"
fi

export NATIVE_FLAGS="-D_UUID_T -D__GETHOSTUUID_H -D_DARWIN_C_SOURCE -D_FILE_OFFSET_BITS=64"
export HOSTCFLAGS="-I${MACOS_HEADERS} -I${LIBELF_INCLUDE} ${NATIVE_FLAGS}"

# 4. Helper Functions
# -------------------
ensure_mounted() {
	if ! mount | grep -q "${MOUNT_POINT}"; then
		echo "Error: Volume not mounted. Run 'km mount' first."
		exit 1
	fi
}
