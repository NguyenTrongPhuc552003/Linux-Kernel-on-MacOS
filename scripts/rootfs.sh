#!/bin/bash
# scripts/rootfs.sh
# Handles creation and packaging of the minimal Debian root filesystem (Initramfs).

# Requires: DEBOOTSTRAP_PATH, ROOTFS_DIR, INITRAMFS_ARCHIVE, DEBIAN_MIRROR, TARGET_ARCH

# ─────────────────────────────────────────────────────────────
# Internal Helper: Check debootstrap status
# ─────────────────────────────────────────────────────────────
_is_rootfs_ready() {
	# We consider the rootfs ready if the main debootstrap directory and the init file exist.
	[ -d "${ROOTFS_DIR}/debootstrap" ] && [ -f "${ROOTFS_DIR}/debootstrap/debootstrap" ]
}

# ─────────────────────────────────────────────────────────────
# 1. Create Minimal Debian Root Filesystem (First Stage)
# ─────────────────────────────────────────────────────────────
create_minimal_rootfs() {
	ensure_mounted # Ensure the target volume is mounted

	# Override the target architecture with riscv64 for RISC-V builds
	if [ "$TARGET_ARCH" = "riscv" ]; then
		TARGET_ARCH="riscv64"
	fi

	# 1. Check if rootfs is already prepared
	if _is_rootfs_ready; then
		echo -e "  [${YELLOW}INFO${NC}] Root filesystem directory already exists (${ROOTFS_DIR})."
		echo -n "  [?] Do you want to clean and restart the debootstrap process (y/N)? "
		read -r response
		if ! [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
			echo -e "  [${YELLOW}SKIP${NC}] Skipping debootstrap setup."
			return 0
		fi
	fi

	# 2. Cleanup and setup output directory (Only executed if not skipping)
	echo -e "  [${YELLOW}ROOTFS${NC}] Preparing minimal root filesystem for ARCH=${TARGET_ARCH}..."

	# Use sudo if the cleanup might involve removing files owned by root/fakeroot
	if sudo rm -rf "${ROOTFS_DIR}"; then
		echo -e "  [${GREEN}CLEAN${NC}] Cleaned old rootfs."
	else
		echo -e "  [${RED}FAIL${NC}] Could not clean old rootfs. Check permissions."
		exit 1
	fi

	mkdir -p "${ROOTFS_DIR}"

	# 3. Execute the first stage (cross-bootstrap)
	# The --foreign flag allows bootstrapping a target arch on a foreign host (macOS)
	echo -e "  [${YELLOW}DEBOOTSTRAP${NC}] Starting debootstrap first stage..."
	if sudo DEBOOTSTRAP_DIR="${DEBOOTSTRAP_PATH}" fakeroot "${DEBOOTSTRAP_PATH}/debootstrap" --foreign \
		--arch="${TARGET_ARCH}" \
		--no-check-sig \
		stable \
		"${ROOTFS_DIR}" \
		"${DEBIAN_MIRROR}"; then
		echo -e "  [${GREEN}SUCCESS${NC}] debootstrap first stage complete."
	else
		echo -e "  [${RED}FAIL${NC}] debootstrap first stage failed."
		exit 1
	fi

	# 4. Create the final 'init' script for the kernel to execute
	# This must be done *after* debootstrap creates the initial filesystem structure.
	echo -e "  [${YELLOW}DEBOOTSTRAP${NC}] Creating /init script for second stage..."
	cat <<'EOF' >"${ROOTFS_DIR}/init"
#!/bin/sh
# This script will be executed by the kernel after booting

/debootstrap/debootstrap --second-stage # Executes the second stage inside QEMU

mount -t proc none /proc
mount -t sysfs none /sys

echo "--- Minimal Debian initramfs booted successfully ---"

# Start a shell or init process (if second stage didn't provide one)
exec /bin/sh

EOF
	chmod +x "${ROOTFS_DIR}/init"
}

# ─────────────────────────────────────────────────────────────
# 2. Package Root Filesystem (CPIO)
# ─────────────────────────────────────────────────────────────
package_initramfs() {
	create_minimal_rootfs # Ensure the rootfs is built first

	# Check if CPIO archive already exists
	if [ -f "${INITRAMFS_ARCHIVE}" ]; then
		echo -e "  [${YELLOW}CPIO${NC}] Initramfs archive already exists: ${INITRAMFS_ARCHIVE}"
		echo -n "  [?] Rebuild CPIO archive (y/N)? "
		read -r response

		# Default is No (N). Only rebuild if user explicitly enters 'y' or 'Y'.
		if ! [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
			echo -e "  [${YELLOW}SKIP${NC}] Using existing Initramfs archive."
			return 0
		fi
	fi

	# If file did not exist OR user confirmed rebuild, proceed
	echo -e "  [${YELLOW}CPIO${NC}] Creating Initramfs CPIO archive..."

	# Ensure init is present before packaging
	if [ ! -f "${ROOTFS_DIR}/init" ]; then
		echo -e "  [${RED}ERROR${NC}] /init script missing. Rootfs setup incomplete."
		exit 1
	fi

	# Use find and cpio to create the archive
	if (cd "${ROOTFS_DIR}" && find . -print0 | cpio --null -ov --format=newc >"${INITRAMFS_ARCHIVE}"); then
		echo -e "  [${GREEN}SUCCESS${NC}] Initramfs archive created: ${INITRAMFS_ARCHIVE}"
	else
		echo -e "  [${RED}FAIL${NC}] Failed to create Initramfs archive."
		exit 1
	fi
}
